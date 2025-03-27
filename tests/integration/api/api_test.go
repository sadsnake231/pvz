//go:build integration
// +build integration

package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/authrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/orderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/reportrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/userorderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/router"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/authstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/orderstorage"
	reportorder "gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/reportorderstorage"
	userorder "gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/userorderstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/tests/integration/testutils"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOrderLifecycle(t *testing.T) {
	ctx := context.Background()
	container, db := testutils.SetupTestDB(ctx, t)
	defer testutils.TeardownTestDB(ctx, t, container, db)

	router := setupRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("order lifecycle", func(t *testing.T) {
		user := map[string]interface{}{
			"email":    "dev@ozon.ru",
			"password": "pwddddddd",
		}
		resp := doRequest(t, ts, "POST", "/users/signup", user, "")
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		resp = doRequest(t, ts, "POST", "/users/login", user, "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		token := getCookieValue(t, resp, "jwt")

		order := map[string]interface{}{
			"id":           "order123",
			"recipient_id": "user1",
			"expiry":       time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			"base_price":   1000,
			"weight":       10,
			"packaging":    "коробка",
		}
		resp = doRequest(t, ts, "POST", "/orders", order, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		resp = doRequest(t, ts, "GET", "/reports/user1/orders", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		orders := response["orders"].([]interface{})
		assert.Len(t, orders, 1)

		resp = doRequest(t, ts, "DELETE", "/orders/order123/return", nil, token)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	})
}

func TestAuthHandlers(t *testing.T) {
	ctx := context.Background()
	container, db := testutils.SetupTestDB(ctx, t)
	defer testutils.TeardownTestDB(ctx, t, container, db)

	router := setupRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("duplicate registration", func(t *testing.T) {
		user := map[string]interface{}{
			"email":    "duplicate@example.com",
			"password": "Password123!",
		}

		resp := doRequest(t, ts, "POST", "/users/signup", user, "")
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		resp = doRequest(t, ts, "POST", "/users/signup", user, "")
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		assert.Contains(t, response["error"], "пользователь с таким email уже зарегистрирован")
	})
}

func TestOrderHandlers(t *testing.T) {
	ctx := context.Background()
	container, db := testutils.SetupTestDB(ctx, t)
	defer testutils.TeardownTestDB(ctx, t, container, db)

	router := setupRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	user := map[string]interface{}{
		"email":    "orderuser@example.com",
		"password": "SecurePass123!",
	}
	resp := doRequest(t, ts, "POST", "/users/signup", user, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp = doRequest(t, ts, "POST", "/users/login", user, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	token := getCookieValue(t, resp, "jwt")

	orders := []map[string]interface{}{
		{
			"id":           "order11",
			"recipient_id": "user1",
			"expiry":       time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			"base_price":   100,
			"weight":       2,
			"packaging":    "пакет",
		},
		{
			"id":           "order12",
			"recipient_id": "user1",
			"expiry":       time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			"base_price":   200,
			"weight":       5,
			"packaging":    "коробка",
		},
	}

	for _, order := range orders {
		resp := doRequest(t, ts, "POST", "/orders", order, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		time.Sleep(1 * time.Second)
	}

	t.Run("get user orders with filters", func(t *testing.T) {
		resp := doRequest(t, ts, "GET", "/reports/user1/orders?limit=1&status=stored", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

		orders := response["orders"].([]interface{})
		assert.Len(t, orders, 1)
	})

	t.Run("issue-refund and get refunded orders", func(t *testing.T) {
		issueRequest := map[string]interface{}{
			"command":   "issue",
			"user_id":   "user1",
			"order_ids": []string{"order11"},
		}

		resp = doRequest(t, ts, "PUT", "/actions/issues_refunds", issueRequest, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		refundRequest := map[string]interface{}{
			"command":   "refund",
			"user_id":   "user1",
			"order_ids": []string{"order11"},
		}
		resp = doRequest(t, ts, "PUT", "/actions/issues_refunds", refundRequest, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp = doRequest(t, ts, "GET", "/reports/refunded", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		orders := response["orders"].([]interface{})
		assert.Len(t, orders, 1)
		assert.Equal(t, "order11", orders[0].(map[string]interface{})["id"])
	})

	t.Run("get order history", func(t *testing.T) {
		resp := doRequest(t, ts, "GET", "/reports/history?limit=2", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		orders := response["orders"].([]interface{})
		assert.Len(t, orders, 2)

		firstTime, _ := time.Parse(time.RFC3339, orders[0].(map[string]interface{})["stored_at"].(string))
		secondTime, _ := time.Parse(time.RFC3339, orders[1].(map[string]interface{})["stored_at"].(string))
		assert.True(t, firstTime.Before(secondTime))
	})

	orders = []map[string]interface{}{
		{
			"id":           "order21",
			"recipient_id": "user1",
			"expiry":       time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			"base_price":   100,
			"weight":       2,
			"packaging":    "пакет",
		},
		{
			"id":           "order22",
			"recipient_id": "user1",
			"expiry":       time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			"base_price":   200,
			"weight":       5,
			"packaging":    "коробка",
		},
	}

	for _, order := range orders {
		resp := doRequest(t, ts, "POST", "/orders", order, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		time.Sleep(1 * time.Second)
	}

	t.Run("issue and refund orders", func(t *testing.T) {
		issueRequest := map[string]interface{}{
			"command":   "issue",
			"user_id":   "user1",
			"order_ids": []string{"order21", "order22"},
		}

		resp := doRequest(t, ts, "PUT", "/actions/issues_refunds", issueRequest, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		assert.Len(t, response["processed_order_ids"], 2)

		refundRequest := map[string]interface{}{
			"command":   "refund",
			"user_id":   "user1",
			"order_ids": []string{"order21", "invalid-order"},
		}

		resp = doRequest(t, ts, "PUT", "/actions/issues_refunds", refundRequest, token)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	})
}

func TestErrorScenarios(t *testing.T) {
	ctx := context.Background()
	container, db := testutils.SetupTestDB(ctx, t)
	defer testutils.TeardownTestDB(ctx, t, container, db)

	router := setupRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("unauthorized access", func(t *testing.T) {
		resp := doRequest(t, ts, "GET", "/reports/refunded", nil, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	user := map[string]interface{}{
		"email":    "test@example.com",
		"password": "testttttt",
	}
	resp := doRequest(t, ts, "POST", "/users/signup", user, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp = doRequest(t, ts, "POST", "/users/login", user, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	token := getCookieValue(t, resp, "jwt")

	t.Run("invalid order creation after login", func(t *testing.T) {
		invalidOrder := map[string]interface{}{
			"id":    "invalid-order",
			"price": 100,
		}

		resp := doRequest(t, ts, "POST", "/orders", invalidOrder, token)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		assert.Contains(t, response["error"], "тело запроса содержит ошибки")
	})

	t.Run("invalid token", func(t *testing.T) {
		resp := doRequest(t, ts, "GET", "/reports/refunded", nil, "invalid-token")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid refund command", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"command": "invalid",
			"user_id": "user-1",
		}

		resp := doRequest(t, ts, "PUT", "/actions/issues_refunds", invalidRequest, token)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func setupRouter(db *pgxpool.Pool) *gin.Engine {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Не смог запустить логгер: %v", err)
	}
	sugarLogger := logger.Sugar()
	defer sugarLogger.Sync()

	orderStorage := orderstorage.NewOrderStorage(db)
	userOrderStorage := userorder.NewUserOrderStorage(db)
	reportOrderStorage := reportorder.NewReportOrderStorage(db)
	authStorage := authstorage.NewAuthStorage(db)

	orderRepo := orderrepo.NewOrderRepository(orderStorage, sugarLogger)
	userRepo := userorderrepo.NewUserOrderRepository(userOrderStorage, sugarLogger)
	reportRepo := reportrepo.NewReportRepository(reportOrderStorage, sugarLogger)
	authRepo := authrepo.NewAuthRepository(authStorage, sugarLogger)

	orderService := service.NewOrderService(orderRepo, userRepo, reportRepo)
	authService := service.NewAuthService(authRepo)

	apiHandler := api.NewAPIHandler(orderService)
	authHandler := api.NewAuthHandler(authService, sugarLogger)

	return router.SetupRouter(apiHandler, authHandler, sugarLogger)
}

func doRequest(t *testing.T, ts *httptest.Server, method, path string, body interface{}, token string) *http.Response {
	var reqBody bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&reqBody).Encode(body))
	}

	req, err := http.NewRequest(method, ts.URL+path, &reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func getCookieValue(t *testing.T, resp *http.Response, name string) string {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	t.Fatalf("Куки %s не найден", name)
	return ""
}

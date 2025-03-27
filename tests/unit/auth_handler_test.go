package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"

	"go.uber.org/zap"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthService) Login(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthService) GenerateToken(email string) (string, error) {
	args := m.Called(email)
	return args.String(0), args.Error(1)
}

func TestAuthHandler_Signup_Success(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "validpassword",
	}
	mockService.On("Register", mock.Anything, user).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/signup",
		bytes.NewBufferString(`{
			"email": "test@example.com",
			"password": "validpassword"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Signup(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "пользователь зарегистрирован")
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Signup_ValidationError(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/signup",
		bytes.NewBufferString(`{
			"email": "invalid-email",
			"password": "short"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), domain.ErrWrongJSON.Error())
}

func TestAuthHandler_Signup_UserAlreadyExists(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "validpassword",
	}
	mockService.On("Register", mock.Anything, user).Return(domain.ErrUserAlreadyExists)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/signup",
		bytes.NewBufferString(`{
			"email": "test@example.com",
			"password": "validpassword"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Signup(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), domain.ErrUserAlreadyExists.Error())
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "validpassword",
	}
	mockService.On("Login", mock.Anything, user).Return(nil)
	mockService.On("GenerateToken", user.Email).Return("test-token", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/login",
		bytes.NewBufferString(`{
			"email": "test@example.com",
			"password": "validpassword"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "вход успешен")

	cookies := w.Result().Cookies()
	assert.Equal(t, "jwt", cookies[0].Name)
	assert.Equal(t, "test-token", cookies[0].Value)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	mockService.On("Login", mock.Anything, user).Return(domain.ErrInvalidCredentials)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/login",
		bytes.NewBufferString(`{
			"email": "test@example.com",
			"password": "wrongpassword"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), domain.ErrInvalidCredentials.Error())
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_TokenGenerationError(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "validpassword",
	}
	mockService.On("Login", mock.Anything, user).Return(nil)
	mockService.On("GenerateToken", user.Email).Return("", domain.ErrTokenGeneration)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/login",
		bytes.NewBufferString(`{
			"email": "test@example.com",
			"password": "validpassword"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), domain.ErrTokenGeneration.Error())
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Signup_EmptyJSON(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/signup",
		bytes.NewBufferString(`{}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "тело запроса содержит ошибки")
}

func TestAuthHandler_Login_Success_JSONStructure(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	user := &domain.User{
		Email:    "test@example.com",
		Password: "validpassword",
	}
	mockService.On("Login", mock.Anything, user).Return(nil)
	mockService.On("GenerateToken", user.Email).Return("test-token", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/login",
		bytes.NewBufferString(`{
			"email": "test@example.com",
			"password": "validpassword"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "вход успешен", response["message"])
}

func TestAuthHandler_Signup_InvalidJSONStructure(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/signup",
		bytes.NewBufferString(`{
			"email": "test@example.com"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Signup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "тело запроса содержит ошибки")
}

func TestAuthHandler_Login_InvalidJSONStructure(t *testing.T) {
	mockService := new(MockAuthService)
	logger := zap.NewNop().Sugar()
	handler := api.NewAuthHandler(mockService, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/users/login",
		bytes.NewBufferString(`{
			"email": "test@example.com"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Field validation for 'Password' failed")
}

package router

import (
	"net/http"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/middleware"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func SetupRouter(apiHandler *api.APIHandler, authHandler *api.AuthHandler, logger *zap.SugaredLogger, auditPipeline *audit.Pipeline) *gin.Engine {
	router := gin.Default()

	router.Use(middleware.AuditMiddleware(auditPipeline))
	router.Use(middleware.MetricsMiddleware())

	orders := router.Group("/orders")
	orders.Use(middleware.AuthMiddleware())
	{
		orders.POST("", apiHandler.AcceptOrder)
		orders.DELETE("/:id/return", apiHandler.ReturnOrder)
	}

	actions := router.Group("/actions")
	actions.Use(middleware.AuthMiddleware())
	{
		actions.PUT("/issues_refunds", apiHandler.IssueRefundOrders)
	}

	reports := router.Group("/reports")
	reports.Use(middleware.AuthMiddleware())
	{
		reports.GET("/:user_id/orders", apiHandler.GetUserOrders)
		reports.GET("/refunded", apiHandler.GetRefundedOrders)
		reports.GET("/history", apiHandler.GetOrderHistory)
		reports.GET("/:user_id/orders/active", apiHandler.GetUserActiveOrders)
		reports.GET("/active", apiHandler.GetAllActiveOrders)
		reports.GET("/history/v2", apiHandler.GetOrderHistoryV2)
	}

	users := router.Group("/users")
	{
		users.POST("/signup", authHandler.Signup)
		users.POST("/login", authHandler.Login)
	}

	health := router.Group("/health")
	{
		health.GET("", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}

	metrics := router.Group("/metrics")
	{
		metrics.GET("", apiHandler.GetMetrics)
	}

	return router
}

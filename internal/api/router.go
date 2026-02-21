package api

import (
	"awesomeProject/pkg/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter creates and configures the Gin router.
func NewRouter(h *UserHandler) *gin.Engine {
	r := gin.Default()

	// Middleware
	r.Use(middleware.CorrelationID())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// User routes
	r.POST("/users", h.CreateUser)
	r.PUT("/users/:id", h.UpdateUser)
	r.GET("/users/:id", h.GetUser)
	r.GET("/users", h.ListUsers)

	return r
}

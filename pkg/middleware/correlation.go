package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CorrelationIDHeader = "X-Correlation-ID"
const CorrelationIDKey = "correlation_id"

// CorrelationID is a Gin middleware that extracts or generates a correlation ID.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		c.Set(CorrelationIDKey, correlationID)
		c.Header(CorrelationIDHeader, correlationID)

		c.Next()
	}
}

// GetCorrelationID retrieves the correlation ID from the Gin context.
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(CorrelationIDKey); exists {
		return id.(string)
	}
	return uuid.New().String()
}

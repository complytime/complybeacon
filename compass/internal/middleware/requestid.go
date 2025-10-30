package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "requestId"
const requestIDHeader = "X-Request-Id"

// RequestID ensures every request has a request ID.
// - Reads X-Request-Id if provided, otherwise generates a UUID.
// - Stores it in gin.Context and echoes it in the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(requestIDKey, rid)
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Next()
	}
}

// GetRequestID returns the request ID stored in the context (if any).
func GetRequestID(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

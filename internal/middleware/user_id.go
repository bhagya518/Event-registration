package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// userIDKey is the type used for the context key to avoid collisions.
type userIDKey string

const UserIDKey userIDKey = "userID"

// UserIDMiddleware extracts the X-User-ID header, validates it, and stores it
// in the request context under UserIDKey.
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":  "missing_user_id",
				"detail": "X-User-ID header is required",
			})
			c.Abort()
			return
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "invalid_user_id",
				"detail": "X-User-ID must be a positive integer",
			})
			c.Abort()
			return
		}

		// Store userID in the context
		c.Set(string(UserIDKey), userID)
		c.Next()
	}
}

// GetUserIDFromContext retrieves the user ID from the Gin context.
// Returns false if the user ID is not present.
func GetUserIDFromContext(c *gin.Context) (int, bool) {
	userID, exists := c.Get(string(UserIDKey))
	if !exists {
		return 0, false
	}
	uid, ok := userID.(int)
	return uid, ok
}

// ContextWithUserID returns a new context with the user ID set.
// This can be useful for passing the user ID to service layers that accept context.Context.
func ContextWithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// UserIDFromContext extracts the user ID from a context.Context.
// Returns false if the user ID is not present.
func UserIDFromContext(ctx context.Context) (int, bool) {
	userID := ctx.Value(UserIDKey)
	if userID == nil {
		return 0, false
	}
	uid, ok := userID.(int)
	return uid, ok
}

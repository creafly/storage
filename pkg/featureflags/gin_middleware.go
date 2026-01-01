package featureflags

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GinMiddleware(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := NewContext()

		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(string); ok {
				ctx.WithUserID(id)
			}
		}

		if tenantID, exists := c.Get("tenant_id"); exists {
			if id, ok := tenantID.(string); ok {
				ctx.WithTenantID(id)
			}
		}

		if sessionID, exists := c.Get("session_id"); exists {
			if id, ok := sessionID.(string); ok {
				ctx.WithSessionID(id)
			}
		}

		ctx.WithRemoteAddr(c.ClientIP())

		c.Set("feature_context", ctx)

		c.Next()
	}
}

func GetContextFromGin(c *gin.Context) *Context {
	if ctx, exists := c.Get("feature_context"); exists {
		if ffCtx, ok := ctx.(*Context); ok {
			return ffCtx
		}
	}
	return NewContext()
}

func RequireFeature(client *Client, featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := GetContextFromGin(c)
		if !client.IsEnabled(featureName, ctx) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "feature_disabled",
				"message": "This feature is currently disabled",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireGlobalFeature(client *Client, featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !client.IsGlobalEnabled(featureName) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "feature_disabled",
				"message": "This feature is currently disabled",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireTenantFeature(client *Client, featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := GetContextFromGin(c)
		if !client.IsTenantEnabled(featureName, ctx) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "feature_disabled",
				"message": "This feature is currently disabled for your organization",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func BlockIfGlobalEnabled(client *Client, featureName string, errorMessage string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client.IsGlobalEnabled(featureName) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "feature_blocked",
				"message": errorMessage,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

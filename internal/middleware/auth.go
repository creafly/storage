package middleware

import (
	"net/http"
	"strings"

	sharedmw "github.com/creafly/middleware"
	"github.com/creafly/storage/internal/i18n"
	"github.com/creafly/storage/internal/infra/client"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetLocale(c *gin.Context) string {
	return sharedmw.GetLocale(c)
}

func AuthMiddleware(identityClient *client.IdentityClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := GetLocale(c)
		messages := i18n.GetMessages(locale)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": messages.Errors.Unauthorized})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": messages.Errors.Unauthorized})
			return
		}

		verifyResp, err := identityClient.VerifyToken(c.Request.Context(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": messages.Errors.Unauthorized})
			return
		}

		if !verifyResp.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": messages.Errors.Unauthorized})
			return
		}

		if verifyResp.IsBlocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":       messages.Errors.UserBlocked,
				"blockReason": verifyResp.BlockReason,
				"blockedAt":   verifyResp.BlockedAt,
			})
			return
		}

		c.Set("userID", verifyResp.UserID)
		c.Set("email", verifyResp.Email)

		if tenantIDHeader := c.GetHeader("X-Tenant-ID"); tenantIDHeader != "" {
			if tenantID, err := uuid.Parse(tenantIDHeader); err == nil {
				c.Set("tenantID", tenantID)
			}
		} else if verifyResp.TenantID != nil {
			c.Set("tenantID", *verifyResp.TenantID)
		}

		c.Next()
	}
}

func TenantValidatorMiddleware(identityClient *client.IdentityClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := GetLocale(c)
		messages := i18n.GetMessages(locale)

		tenantID, exists := c.Get("tenantID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
			return
		}

		authHeader := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		valid, err := identityClient.ValidateTenantAccess(c.Request.Context(), tokenString, tenantID.(uuid.UUID))
		if err != nil || !valid {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
			return
		}

		c.Next()
	}
}

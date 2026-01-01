package middleware

import "github.com/gin-gonic/gin"

const defaultLocale = "en-US"

func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := c.GetHeader("Accept-Language")
		if locale == "" {
			locale = defaultLocale
		}
		c.Set("locale", locale)
		c.Next()
	}
}

func GetLocale(c *gin.Context) string {
	if locale, exists := c.Get("locale"); exists {
		return locale.(string)
	}
	return defaultLocale
}

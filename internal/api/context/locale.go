package context

import (
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	i18npkg "github.com/xzzpig/rclone-sync/internal/i18n"
)

// GinContextKeyLocalizer is the key for storing Localizer in Gin context
const GinContextKeyLocalizer = "localizer"

// LocaleMiddleware parses Accept-Language header and stores Localizer in both contexts
func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		locale := i18npkg.ParseLocale(lang)
		localizer := i18npkg.NewLocalizer(locale)

		// 1. Store in Gin context (for handlers)
		c.Set("locale", locale)
		c.Set(GinContextKeyLocalizer, localizer)

		// 2. Store in context.Context (for business logic / services)
		// 通过修改 c.Request 的 Context 来传递到业务层
		ctx := c.Request.Context()
		ctx = i18npkg.WithLocalizer(ctx, localizer)
		ctx = i18npkg.WithLocale(ctx, locale)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetLocalizer retrieves the Localizer from Gin context
func GetLocalizer(c *gin.Context) *i18n.Localizer {
	if localizer, exists := c.Get(GinContextKeyLocalizer); exists {
		return localizer.(*i18n.Localizer)
	}
	// Fallback: try to get from request context
	return i18npkg.LocalizerFromContext(c.Request.Context())
}

// I18nErrorMiddleware 自动处理 I18nError 类型的错误
// 使用 Gin 的错误处理机制：c.Error(err)
func I18nErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) == 0 {
			return
		}

		// 获取 localizer
		localizer := GetLocalizer(c)

		// 处理最后一个错误
		lastErr := c.Errors.Last()
		if lastErr == nil {
			return
		}

		// 检查是否为 I18nError
		if i18nErr, ok := i18npkg.IsI18nError(lastErr.Err); ok {
			// 翻译错误消息
			translatedMsg := i18nErr.Translate(localizer)

			// 返回 JSON 响应
			c.JSON(i18nErr.StatusCode, gin.H{
				"error":   translatedMsg,
				"code":    i18nErr.MsgID,
				"success": false,
			})
			return
		}

		// 非 I18nError，返回通用错误
		c.JSON(500, gin.H{
			"error":   i18npkg.T(localizer, i18npkg.ErrGeneric),
			"code":    "internal_error",
			"success": false,
		})
	}
}

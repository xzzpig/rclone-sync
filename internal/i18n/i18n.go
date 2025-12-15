package i18n

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

var bundle *i18n.Bundle

// Init 初始化 i18n bundle
// 应该在应用启动时调用
func Init() error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// 从嵌入文件加载
	if _, err := bundle.LoadMessageFileFS(localeFS, "locales/en.toml"); err != nil {
		return fmt.Errorf("failed to load en.toml: %w", err)
	}
	if _, err := bundle.LoadMessageFileFS(localeFS, "locales/zh-CN.toml"); err != nil {
		return fmt.Errorf("failed to load zh-CN.toml: %w", err)
	}

	return nil
}

// NewLocalizer creates a new localizer for the given language
func NewLocalizer(lang string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle, lang)
}

// ParseLocale normalizes a language string to a supported locale
func ParseLocale(s string) string {
	if strings.HasPrefix(s, "zh") {
		return "zh-CN"
	}
	return "en"
}

// T translates a message with the given localizer
func T(localizer *i18n.Localizer, msgID string) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: msgID,
	})
	if err != nil {
		return msgID // fallback to key
	}
	return msg
}

// TWithData translates a message with template data
func TWithData(localizer *i18n.Localizer, msgID string, data map[string]interface{}) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
	})
	if err != nil {
		return msgID
	}
	return msg
}

// TPlural translates a message with plural support
func TPlural(localizer *i18n.Localizer, msgID string, count int, data map[string]interface{}) string {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["Count"] = count
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
		PluralCount:  count,
	})
	if err != nil {
		return msgID
	}
	return msg
}

// ========== context.Context 相关函数 ==========

// contextKey 是 context.Context 中存储值的键类型
type contextKey string

const (
	// ContextKeyLocalizer 是 context.Context 中 Localizer 的键
	ContextKeyLocalizer contextKey = "i18n.localizer"
	// ContextKeyLocale 是 context.Context 中 locale 字符串的键
	ContextKeyLocale contextKey = "i18n.locale"
)

// WithLocalizer 将 Localizer 存入 context.Context
func WithLocalizer(ctx context.Context, localizer *i18n.Localizer) context.Context {
	return context.WithValue(ctx, ContextKeyLocalizer, localizer)
}

// WithLocale 将 locale 字符串存入 context.Context
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, ContextKeyLocale, locale)
}

// LocalizerFromContext 从 context.Context 获取 Localizer
// 如果不存在，返回英文 Localizer
func LocalizerFromContext(ctx context.Context) *i18n.Localizer {
	if localizer, ok := ctx.Value(ContextKeyLocalizer).(*i18n.Localizer); ok {
		return localizer
	}
	return NewLocalizer("en")
}

// LocaleFromContext 从 context.Context 获取 locale 字符串
// 如果不存在，返回 "en"
func LocaleFromContext(ctx context.Context) string {
	if locale, ok := ctx.Value(ContextKeyLocale).(string); ok {
		return locale
	}
	return "en"
}

// Ctx 是在业务逻辑中使用的便捷翻译函数
// 直接从 context.Context 获取 Localizer 并翻译消息
func Ctx(ctx context.Context, msgID string) string {
	return T(LocalizerFromContext(ctx), msgID)
}

// CtxWithData 是在业务逻辑中使用的带数据便捷翻译函数
func CtxWithData(ctx context.Context, msgID string, data map[string]interface{}) string {
	return TWithData(LocalizerFromContext(ctx), msgID, data)
}

// CtxPlural 是在业务逻辑中使用的复数便捷翻译函数
func CtxPlural(ctx context.Context, msgID string, count int, data map[string]interface{}) string {
	return TPlural(LocalizerFromContext(ctx), msgID, count, data)
}

// ========== I18nError 错误类型 ==========

// I18nError 是一个可翻译的错误类型
// 用于在业务逻辑层返回需要翻译的错误
type I18nError struct {
	// MsgID 是翻译消息的键
	MsgID string
	// Data 是翻译模板的数据（可选）
	Data map[string]interface{}
	// StatusCode 是 HTTP 状态码（默认 400）
	StatusCode int
	// Cause 是原始错误（可选）
	Cause error
}

// Error 实现 error 接口
// 返回消息 ID（用于日志等场景）
func (e *I18nError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.MsgID, e.Cause)
	}
	return e.MsgID
}

// Unwrap 返回原始错误
func (e *I18nError) Unwrap() error {
	return e.Cause
}

// Translate 使用给定的 localizer 翻译错误消息
func (e *I18nError) Translate(localizer *i18n.Localizer) string {
	if e.Data != nil {
		return TWithData(localizer, e.MsgID, e.Data)
	}
	return T(localizer, e.MsgID)
}

// TranslateCtx 使用 context 中的 localizer 翻译错误消息
func (e *I18nError) TranslateCtx(ctx context.Context) string {
	return e.Translate(LocalizerFromContext(ctx))
}

// NewI18nError 创建一个新的 I18nError
func NewI18nError(msgID string) *I18nError {
	return &I18nError{
		MsgID:      msgID,
		StatusCode: 400,
	}
}

// NewI18nErrorWithData 创建一个带数据的 I18nError
func NewI18nErrorWithData(msgID string, data map[string]interface{}) *I18nError {
	return &I18nError{
		MsgID:      msgID,
		Data:       data,
		StatusCode: 400,
	}
}

// WithStatus 设置 HTTP 状态码
func (e *I18nError) WithStatus(code int) *I18nError {
	e.StatusCode = code
	return e
}

// WithCause 设置原始错误
func (e *I18nError) WithCause(err error) *I18nError {
	e.Cause = err
	return e
}

// WithData 设置翻译数据
func (e *I18nError) WithData(data map[string]interface{}) *I18nError {
	e.Data = data
	return e
}

// 常用错误构造函数

// ErrNotFoundI18n 返回 404 错误
func ErrNotFoundI18n(msgID string) *I18nError {
	return NewI18nError(msgID).WithStatus(404)
}

// ErrBadRequestI18n 返回 400 错误
func ErrBadRequestI18n(msgID string) *I18nError {
	return NewI18nError(msgID).WithStatus(400)
}

// ErrInternalI18n 返回 500 错误
func ErrInternalI18n(msgID string) *I18nError {
	return NewI18nError(msgID).WithStatus(500)
}

// ErrUnauthorizedI18n 返回 401 错误
func ErrUnauthorizedI18n(msgID string) *I18nError {
	return NewI18nError(msgID).WithStatus(401)
}

// IsI18nError 检查错误是否为 I18nError
func IsI18nError(err error) (*I18nError, bool) {
	var i18nErr *I18nError
	if errors.As(err, &i18nErr) {
		return i18nErr, true
	}
	return nil, false
}

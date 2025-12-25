// Package i18n provides internationalization support for the application.
package i18n

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

var bundle *i18n.Bundle

// Init initializes the i18n bundle.
// Should be called when the application starts.
func Init() error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Load from embedded files
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

// ========== context.Context related functions ==========

// contextKey is the type for keys used to store values in context.Context
type contextKey string

const (
	// ContextKeyLocalizer is the key for Localizer in context.Context
	ContextKeyLocalizer contextKey = "i18n.localizer"
	// ContextKeyLocale is the key for the locale string in context.Context
	ContextKeyLocale contextKey = "i18n.locale"
)

// WithLocalizer stores a Localizer in context.Context
func WithLocalizer(ctx context.Context, localizer *i18n.Localizer) context.Context {
	return context.WithValue(ctx, ContextKeyLocalizer, localizer)
}

// WithLocale stores a locale string in context.Context
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, ContextKeyLocale, locale)
}

// LocalizerFromContext retrieves a Localizer from context.Context
// If not found, returns an English Localizer
func LocalizerFromContext(ctx context.Context) *i18n.Localizer {
	if localizer, ok := ctx.Value(ContextKeyLocalizer).(*i18n.Localizer); ok {
		return localizer
	}
	return NewLocalizer("en")
}

// LocaleFromContext retrieves a locale string from context.Context
// If not found, returns "en"
func LocaleFromContext(ctx context.Context) string {
	if locale, ok := ctx.Value(ContextKeyLocale).(string); ok {
		return locale
	}
	return "en"
}

// Ctx is a convenient translation function for business logic
// It retrieves the Localizer directly from context.Context and translates the message
func Ctx(ctx context.Context, msgID string) string {
	return T(LocalizerFromContext(ctx), msgID)
}

// CtxWithData is a convenient translation function with data for business logic
func CtxWithData(ctx context.Context, msgID string, data map[string]interface{}) string {
	return TWithData(LocalizerFromContext(ctx), msgID, data)
}

// CtxPlural is a convenient plural translation function for business logic
func CtxPlural(ctx context.Context, msgID string, count int, data map[string]interface{}) string {
	return TPlural(LocalizerFromContext(ctx), msgID, count, data)
}

// ========== I18nError error type ==========

// Error is a translatable error type
// Used to return errors that need to be translated in the business logic layer
type Error struct {
	// MsgID is the key for the translated message
	MsgID string
	// Data is the data for the translation template (optional)
	Data map[string]interface{}
	// StatusCode is the HTTP status code (default 400)
	StatusCode int
	// Cause is the original error (optional)
	Cause error
}

// Error implements the error interface
// Returns the message ID (for logging and other scenarios)
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.MsgID, e.Cause)
	}
	return e.MsgID
}

// Unwrap returns the original error
func (e *Error) Unwrap() error {
	return e.Cause
}

// Translate translates the error message using the given localizer
func (e *Error) Translate(localizer *i18n.Localizer) string {
	if e.Data != nil {
		return TWithData(localizer, e.MsgID, e.Data)
	}
	return T(localizer, e.MsgID)
}

// TranslateCtx translates the error message using the localizer from context
func (e *Error) TranslateCtx(ctx context.Context) string {
	return e.Translate(LocalizerFromContext(ctx))
}

// NewI18nError creates a new I18nError
func NewI18nError(msgID string) *Error {
	return &Error{
		MsgID:      msgID,
		StatusCode: 400,
	}
}

// NewI18nErrorWithData creates an I18nError with data
func NewI18nErrorWithData(msgID string, data map[string]interface{}) *Error {
	return &Error{
		MsgID:      msgID,
		Data:       data,
		StatusCode: 400,
	}
}

// WithStatus sets the HTTP status code
func (e *Error) WithStatus(code int) *Error {
	e.StatusCode = code
	return e
}

// WithCause sets the original error
func (e *Error) WithCause(err error) *Error {
	e.Cause = err
	return e
}

// WithData sets the translation data
func (e *Error) WithData(data map[string]interface{}) *Error {
	e.Data = data
	return e
}

// Common error constructors

// ErrNotFoundI18n returns a 404 error
func ErrNotFoundI18n(msgID string) *Error {
	return NewI18nError(msgID).WithStatus(404)
}

// ErrBadRequestI18n returns a 400 error
func ErrBadRequestI18n(msgID string) *Error {
	return NewI18nError(msgID).WithStatus(400)
}

// ErrInternalI18n returns a 500 error
func ErrInternalI18n(msgID string) *Error {
	return NewI18nError(msgID).WithStatus(500)
}

// ErrUnauthorizedI18n returns a 401 error
func ErrUnauthorizedI18n(msgID string) *Error {
	return NewI18nError(msgID).WithStatus(401)
}

// IsI18nError checks if the error is an I18nError
func IsI18nError(err error) (*Error, bool) {
	var i18nErr *Error
	if errors.As(err, &i18nErr) {
		return i18nErr, true
	}
	return nil, false
}

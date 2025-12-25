package graphql

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/zap"

	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

// ConstError is an alias to errs.ConstError for defining sentinel errors in this package.
type ConstError = errs.ConstError

const (
	// ErrInternalServer is returned when an internal server error occurs during GraphQL execution.
	ErrInternalServer ConstError = "internal server error"
)

// errorsLog returns a named logger for the graphql errors package.
func errorsLog() *zap.Logger {
	return logger.Named("api.graphql.errors")
}

// ErrorPresenter translates I18nError to localized GraphQL errors.
func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)

	// Check if it's an I18nError
	if i18nErr, ok := i18n.IsI18nError(err); ok {
		// Get localizer from context
		localizer := i18n.LocalizerFromContext(ctx)

		// Translate the error message
		gqlErr.Message = i18nErr.Translate(localizer)

		// Add error code to extensions
		if gqlErr.Extensions == nil {
			gqlErr.Extensions = make(map[string]any)
		}
		gqlErr.Extensions["code"] = i18nErr.MsgID
	}

	return gqlErr
}

// RecoverFunc is a panic recovery function for GraphQL.
// It logs the panic details including stack trace for debugging purposes.
func RecoverFunc(_ context.Context, p interface{}) error {
	// Capture the stack trace
	stack := string(debug.Stack())

	// Log the panic with full details for debugging
	errorsLog().Error("GraphQL resolver panic recovered",
		zap.Any("panic", p),
		zap.String("panic_details", fmt.Sprintf("%v", p)),
		zap.String("stack", stack),
	)

	// Return a generic error to the client (don't expose internal details)
	return ErrInternalServer
}

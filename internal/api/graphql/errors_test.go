package graphql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/xzzpig/rclone-sync/internal/api/graphql"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

func TestErrorPresenter_RegularError(t *testing.T) {
	ctx := context.Background()
	err := errors.New("regular error message")

	result := graphql.ErrorPresenter(ctx, err)

	require.NotNil(t, result)
	assert.Equal(t, "regular error message", result.Message)
}

func TestErrorPresenter_I18nError(t *testing.T) {
	// Initialize i18n
	err := i18n.Init()
	require.NoError(t, err)

	ctx := context.Background()

	// Create an I18nError
	i18nErr := i18n.NewI18nError("test.error.key")

	result := graphql.ErrorPresenter(ctx, i18nErr)

	require.NotNil(t, result)
	// When no localizer is in context, the message will use the fallback
	assert.NotEmpty(t, result.Message)
	// Extensions should contain the error code
	require.NotNil(t, result.Extensions)
	assert.Equal(t, "test.error.key", result.Extensions["code"])
}

func TestErrorPresenter_I18nErrorWithParams(t *testing.T) {
	// Initialize i18n
	err := i18n.Init()
	require.NoError(t, err)

	ctx := context.Background()

	// Create an I18nError with parameters
	params := map[string]interface{}{
		"Name": "TestItem",
	}
	i18nErr := i18n.NewI18nErrorWithData("error.not.found", params)

	result := graphql.ErrorPresenter(ctx, i18nErr)

	require.NotNil(t, result)
	assert.NotEmpty(t, result.Message)
	require.NotNil(t, result.Extensions)
	assert.Equal(t, "error.not.found", result.Extensions["code"])
}

func TestErrorPresenter_WrappedI18nError(t *testing.T) {
	// Initialize i18n
	err := i18n.Init()
	require.NoError(t, err)

	ctx := context.Background()

	// Create a wrapped I18nError
	i18nErr := i18n.NewI18nError("wrapped.error.key")
	wrappedErr := errors.New("wrapper: " + i18nErr.Error())

	result := graphql.ErrorPresenter(ctx, wrappedErr)

	require.NotNil(t, result)
	// Regular error doesn't have i18n extensions
	assert.Contains(t, result.Message, "wrapper")
}

func TestErrorPresenter_NilExtensions(t *testing.T) {
	// Initialize i18n
	err := i18n.Init()
	require.NoError(t, err)

	ctx := context.Background()

	i18nErr := i18n.NewI18nError("test.key")

	result := graphql.ErrorPresenter(ctx, i18nErr)

	require.NotNil(t, result)
	// Extensions should be created if nil
	require.NotNil(t, result.Extensions)
	assert.Equal(t, "test.key", result.Extensions["code"])
}

func TestErrorPresenter_PreservesGQLErrorPath(t *testing.T) {
	ctx := context.Background()

	gqlErr := &gqlerror.Error{
		Message: "test error",
		Path:    ast.Path{ast.PathName("query"), ast.PathName("field")},
	}

	result := graphql.ErrorPresenter(ctx, gqlErr)

	require.NotNil(t, result)
	assert.Equal(t, "test error", result.Message)
	require.Len(t, result.Path, 2)
}

func TestRecoverFunc_ReturnInternalError(t *testing.T) {
	ctx := context.Background()

	// Simulate a panic recovery
	panicValue := "something went wrong"

	err := graphql.RecoverFunc(ctx, panicValue)

	require.Error(t, err)
	assert.Equal(t, "internal server error", err.Error())
}

func TestRecoverFunc_WithDifferentPanicValues(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name       string
		panicValue interface{}
	}{
		{"string panic", "panic message"},
		{"error panic", errors.New("panic error")},
		{"int panic", 42},
		{"nil panic", nil},
		{"struct panic", struct{ msg string }{"test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := graphql.RecoverFunc(ctx, tc.panicValue)

			require.Error(t, err)
			assert.Equal(t, "internal server error", err.Error())
		})
	}
}

func TestErrorPresenter_WithLocalizer(t *testing.T) {
	// Initialize i18n
	err := i18n.Init()
	require.NoError(t, err)

	// Create context with localizer
	localizer := i18n.NewLocalizer("en")
	ctx := i18n.WithLocalizer(context.Background(), localizer)

	// Create an I18nError
	i18nErr := i18n.NewI18nError("test.key")

	result := graphql.ErrorPresenter(ctx, i18nErr)

	require.NotNil(t, result)
	// The message should be translated (or fallback to key if not found)
	assert.NotEmpty(t, result.Message)
	require.NotNil(t, result.Extensions)
	assert.Equal(t, "test.key", result.Extensions["code"])
}

func TestErrorPresenter_MultipleExtensions(t *testing.T) {
	ctx := context.Background()

	// Create a gqlerror with existing extensions
	gqlErr := &gqlerror.Error{
		Message: "test error",
		Extensions: map[string]interface{}{
			"existingKey": "existingValue",
		},
	}

	result := graphql.ErrorPresenter(ctx, gqlErr)

	require.NotNil(t, result)
	assert.Equal(t, "test error", result.Message)
	// Existing extensions should be preserved
	assert.Equal(t, "existingValue", result.Extensions["existingKey"])
}

func TestErrorPresenter_I18nErrorWithCause(t *testing.T) {
	// Initialize i18n
	initErr := i18n.Init()
	require.NoError(t, initErr)

	ctx := context.Background()

	// Create an I18nError with cause
	cause := errors.New("underlying error")
	i18nErr := i18n.NewI18nError("error.with.cause").WithCause(cause)

	result := graphql.ErrorPresenter(ctx, i18nErr)

	require.NotNil(t, result)
	assert.NotEmpty(t, result.Message)
	require.NotNil(t, result.Extensions)
	assert.Equal(t, "error.with.cause", result.Extensions["code"])
}

func TestErrorPresenter_I18nErrorWithStatus(t *testing.T) {
	// Initialize i18n
	err := i18n.Init()
	require.NoError(t, err)

	ctx := context.Background()

	// Create an I18nError with custom status
	i18nErr := i18n.NewI18nError("not.found.error").WithStatus(404)

	result := graphql.ErrorPresenter(ctx, i18nErr)

	require.NotNil(t, result)
	assert.NotEmpty(t, result.Message)
	require.NotNil(t, result.Extensions)
	assert.Equal(t, "not.found.error", result.Extensions["code"])
}

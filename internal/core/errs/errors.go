// Package errs provides common error types for the application.
package errs

// Sentinel errors for the domain layer.
// These errors should be used to wrap low-level errors (like DB errors)
// so that the upper layers (API/CLI) can handle them appropriately without knowing the implementation details.

// ConstError represents a sentinel error type.
type ConstError string

func (e ConstError) Error() string {
	return string(e)
}

const (
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = ConstError("resource not found")

	// ErrInvalidInput is returned when the input provided is invalid.
	ErrInvalidInput = ConstError("invalid input")

	// ErrSystem is returned when an unexpected system error occurs.
	ErrSystem = ConstError("system error")

	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = ConstError("resource already exists")

	// ErrUnauthorized is returned when the user is not authorized to perform the action.
	ErrUnauthorized = ConstError("unauthorized")

	// ErrValidation is returned when validation of input data fails.
	ErrValidation = ConstError("validation error")
)

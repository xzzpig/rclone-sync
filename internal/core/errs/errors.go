package errs

import "errors"

// Sentinel errors for the domain layer.
// These errors should be used to wrap low-level errors (like DB errors)
// so that the upper layers (API/CLI) can handle them appropriately without knowing the implementation details.

var (
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidInput is returned when the input provided is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrSystem is returned when an unexpected system error occurs.
	ErrSystem = errors.New("system error")

	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrUnauthorized is returned when the user is not authorized to perform the action.
	ErrUnauthorized = errors.New("unauthorized")
)

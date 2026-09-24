package errors

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid request input")
	ErrUnauthorized = errors.New("unauthorized")
)

type BusinessError struct {
	Code    int
	Message string
	Err     error
}

func (e *BusinessError) Error() string { return e.Message }
func (e *BusinessError) Unwrap() error { return e.Err }

// NewValidation builds a client-facing validation error carrying a readable
// message while remaining discoverable via errors.Is(err, ErrInvalidInput).
func NewValidation(message string) *BusinessError {
	return &BusinessError{Message: message, Err: ErrInvalidInput}
}

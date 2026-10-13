package api

import (
	"errors"
	"fmt"
	"net"
)

// ErrRequestPreparation means no HTTP request could be prepared for execution.
var ErrRequestPreparation = errors.New("HTTP request preparation failed")

// ErrInvalidResponse means a successful HTTP response could not be decoded.
// Concrete parser errors remain diagnostic causes, not the response contract.
var ErrInvalidResponse = errors.New("invalid API response")

// RequestExecutionError identifies failure of HTTP client execution. It does
// not by itself establish a network connection failure (e.g. redirect policy).
// Timeout is a supported fact of this specific execution cause.
type RequestExecutionError struct{ Err error }

func (e *RequestExecutionError) Error() string { return fmt.Sprintf("execute HTTP request: %v", e.Err) }
func (e *RequestExecutionError) Unwrap() error { return e.Err }
func (e *RequestExecutionError) Timeout() bool {
	cause, ok := errors.AsType[net.Error](e.Err)
	return ok && cause.Timeout()
}

// ResponseReadError identifies failure to read a successful HTTP response.
// Timeout is a supported fact of this specific read cause.
type ResponseReadError struct{ Err error }

func (e *ResponseReadError) Error() string { return fmt.Sprintf("read HTTP response: %v", e.Err) }
func (e *ResponseReadError) Unwrap() error { return e.Err }
func (e *ResponseReadError) Timeout() bool {
	cause, ok := errors.AsType[net.Error](e.Err)
	return ok && cause.Timeout()
}

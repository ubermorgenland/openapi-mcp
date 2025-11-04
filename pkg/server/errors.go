package server

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"
)

// Error types for structured error handling
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeDatabase     ErrorType = "database"
	ErrorTypeAuth         ErrorType = "authentication"
	ErrorTypeNetwork      ErrorType = "network"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeConflict     ErrorType = "conflict"
)

// ServerError represents a structured error with context
type ServerError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	Timestamp  int64     `json:"timestamp"`
	StackTrace string    `json:"stack_trace,omitempty"`
}

// Error implements the error interface
func (e *ServerError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewError creates a new ServerError with context
func NewError(errType ErrorType, message string, details string) *ServerError {
	return &ServerError{
		Type:      errType,
		Message:   message,
		Details:   details,
		Timestamp: getCurrentTimestamp(),
	}
}

// NewErrorWithContext creates a new ServerError with request context
func NewErrorWithContext(ctx context.Context, errType ErrorType, message string, details string) *ServerError {
	err := NewError(errType, message, details)
	
	// Extract request ID from context if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		err.RequestID = requestID
	}
	
	return err
}

// WithStackTrace adds stack trace information to the error
func (e *ServerError) WithStackTrace() *ServerError {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	e.StackTrace = string(buf[:n])
	return e
}

// LogError logs the error with appropriate level and context
func (e *ServerError) LogError() {
	switch e.Type {
	case ErrorTypeValidation:
		log.Printf("VALIDATION ERROR: %s", e.Error())
	case ErrorTypeAuth:
		log.Printf("AUTH ERROR: %s", e.Error())
	case ErrorTypeDatabase:
		log.Printf("DATABASE ERROR: %s", e.Error())
	case ErrorTypeNetwork:
		log.Printf("NETWORK ERROR: %s", e.Error())
	case ErrorTypeNotFound:
		log.Printf("NOT FOUND: %s", e.Error())
	case ErrorTypeConflict:
		log.Printf("CONFLICT: %s", e.Error())
	default:
		log.Printf("ERROR: %s", e.Error())
	}
	
	if e.StackTrace != "" {
		log.Printf("Stack trace: %s", e.StackTrace)
	}
}

// Wrap wraps a standard error as a ServerError
func Wrap(err error, errType ErrorType, message string) *ServerError {
	if err == nil {
		return nil
	}
	
	return NewError(errType, message, err.Error())
}

// WrapWithContext wraps a standard error as a ServerError with context
func WrapWithContext(ctx context.Context, err error, errType ErrorType, message string) *ServerError {
	if err == nil {
		return nil
	}
	
	return NewErrorWithContext(ctx, errType, message, err.Error())
}

// getCurrentTimestamp returns current Unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// IsType checks if the error is of a specific type
func IsType(err error, errType ErrorType) bool {
	if serverErr, ok := err.(*ServerError); ok {
		return serverErr.Type == errType
	}
	return false
}

// GetType returns the error type if it's a ServerError, otherwise returns ErrorTypeInternal
func GetType(err error) ErrorType {
	if serverErr, ok := err.(*ServerError); ok {
		return serverErr.Type
	}
	return ErrorTypeInternal
}
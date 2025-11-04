package server

import (
	"errors"
	"fmt"
)

var (
	// Common server errors
	ErrUnsupported      = errors.New("not supported")
	ErrResourceNotFound = errors.New("resource not found")
	ErrPromptNotFound   = errors.New("prompt not found")
	ErrToolNotFound     = errors.New("tool not found")

	// Session-related errors
	ErrSessionNotFound              = errors.New("session not found")
	ErrSessionExists                = errors.New("session already exists")
	ErrSessionNotInitialized        = errors.New("session not properly initialized")
	ErrSessionDoesNotSupportTools   = errors.New("session does not support per-session tools")
	ErrSessionDoesNotSupportLogging = errors.New("session does not support setting logging level")

	// Notification-related errors
	ErrNotificationNotInitialized = errors.New("notification channel not initialized")
	ErrNotificationChannelBlocked = errors.New("notification channel full or blocked")

	// Authentication-related errors
	ErrInvalidSessionAuth = errors.New("invalid session authentication")
	ErrExpiredSessionAuth = errors.New("session authentication expired")
	ErrMissingAuth        = errors.New("missing authentication credentials")
	ErrInvalidAuth        = errors.New("invalid authentication credentials")
	ErrSessionTerminated  = errors.New("session terminated")
)

// ErrDynamicPathConfig is returned when attempting to use static path methods with dynamic path configuration
type ErrDynamicPathConfig struct {
	Method string
}

func (e *ErrDynamicPathConfig) Error() string {
	return fmt.Sprintf("%s cannot be used with WithDynamicBasePath. Use dynamic path logic in your router.", e.Method)
}

// AuthErrorType represents different types of authentication errors
type AuthErrorType int

const (
	AuthErrInvalidSession AuthErrorType = iota
	AuthErrExpiredSession
	AuthErrMissingAuth
	AuthErrInvalidAuth
	AuthErrSessionNotFound
	AuthErrSessionTerminated
)

// AuthError represents an authentication-related error with structured information
type AuthError struct {
	Type      AuthErrorType
	Message   string
	SessionID string
	Cause     error
}

func (e *AuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AuthError) Unwrap() error {
	return e.Cause
}

// NewAuthError creates a new authentication error
func NewAuthError(errType AuthErrorType, message string, sessionID string, cause error) *AuthError {
	return &AuthError{
		Type:      errType,
		Message:   message,
		SessionID: sessionID,
		Cause:     cause,
	}
}

// Helper functions for common auth errors
func NewInvalidSessionError(sessionID string) *AuthError {
	return NewAuthError(AuthErrInvalidSession, "invalid session ID", sessionID, nil)
}

func NewExpiredSessionError(sessionID string) *AuthError {
	return NewAuthError(AuthErrExpiredSession, "session authentication expired", sessionID, nil)
}

func NewMissingAuthError() *AuthError {
	return NewAuthError(AuthErrMissingAuth, "missing authentication credentials", "", nil)
}

func NewInvalidAuthError(reason string) *AuthError {
	return NewAuthError(AuthErrInvalidAuth, fmt.Sprintf("invalid authentication: %s", reason), "", nil)
}

func NewSessionNotFoundError(sessionID string) *AuthError {
	return NewAuthError(AuthErrSessionNotFound, "session not found", sessionID, nil)
}

func NewSessionTerminatedError(sessionID string) *AuthError {
	return NewAuthError(AuthErrSessionTerminated, "session terminated", sessionID, nil)
}

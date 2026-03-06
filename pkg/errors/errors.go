package errors

import (
	"fmt"
	"strings"
)

// NewValidationError creates a new validation error.
func NewValidationError(code int, field string, messages ...string) *ValidationError {
	return &ValidationError{Code: code, Field: field, Messages: messages}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, strings.Join(e.Messages, ", "))
}

// NewValidationErrorCollector creates a new validation error collector.
func NewValidationErrorCollector() *ValidationErrorCollector {
	return &ValidationErrorCollector{errors: make([]*ValidationError, 0)}
}

func (c *ValidationErrorCollector) Add(err *ValidationError) *ValidationErrorCollector {
	c.errors = append(c.errors, err)
	return c
}

func (c *ValidationErrorCollector) HasError() bool {
	return len(c.errors) > 0
}

func (c *ValidationErrorCollector) Errors() []*ValidationError {
	return c.errors
}

func (c *ValidationErrorCollector) Error() string {
	var msgs []string
	for _, err := range c.errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, ", ")
}

// NewPermissionError creates a new permission error.
func NewPermissionError(code int, field string, messages ...string) *PermissionError {
	return &PermissionError{Code: code, Field: field, Messages: messages}
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, strings.Join(e.Messages, ", "))
}

// NewPermissionErrorCollector creates a new permission error collector.
func NewPermissionErrorCollector() *PermissionErrorCollector {
	return &PermissionErrorCollector{errors: make([]*PermissionError, 0)}
}

func (c *PermissionErrorCollector) Add(err *PermissionError) *PermissionErrorCollector {
	c.errors = append(c.errors, err)
	return c
}

func (c *PermissionErrorCollector) HasError() bool {
	return len(c.errors) > 0
}

func (c *PermissionErrorCollector) Errors() []*PermissionError {
	return c.errors
}

func (c *PermissionErrorCollector) Error() string {
	var msgs []string
	for _, err := range c.errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, ", ")
}

// NewHTTPError returns a new HTTPError with the given code and message.
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{Code: code, Message: message, StatusCode: code}
}

// NewUnauthorizedHTTPError returns a 401 Unauthorized error.
func NewUnauthorizedHTTPError() *HTTPError {
	return &HTTPError{
		Code:       401,
		Message:    MessageUnauthorized,
		StatusCode: StatusUnauthorized,
	}
}

// NewForbiddenHTTPError returns a 403 Forbidden error.
func NewForbiddenHTTPError() *HTTPError {
	return &HTTPError{
		Code:       403,
		Message:    MessageForbidden,
		StatusCode: StatusForbidden,
	}
}

func (e *HTTPError) Error() string {
	return e.Message
}

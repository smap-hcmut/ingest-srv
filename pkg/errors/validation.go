package errors

import (
	"fmt"
	"strings"
)

// ValidationError is an error with a field and a list of messages.
type ValidationError struct {
	Code     int      `json:"code"`
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// NewValidationError creates a new validation error.
func NewValidationError(code int, field string, messages ...string) *ValidationError {
	return &ValidationError{
		Code:     code,
		Field:    field,
		Messages: messages,
	}
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, strings.Join(e.Messages, ", "))
}

// ValidationErrorCollector collects multiple validation errors.
type ValidationErrorCollector struct {
	errors []*ValidationError
}

// NewValidationErrorCollector creates a new validation error collector.
func NewValidationErrorCollector() *ValidationErrorCollector {
	return &ValidationErrorCollector{
		errors: make([]*ValidationError, 0),
	}
}

// Add adds a new validation error to the collector and returns the collector for chaining.
func (c *ValidationErrorCollector) Add(err *ValidationError) *ValidationErrorCollector {
	c.errors = append(c.errors, err)
	return c
}

// HasError returns true if the collector has any error.
func (c *ValidationErrorCollector) HasError() bool {
	return len(c.errors) > 0
}

// Errors returns the list of errors.
func (c *ValidationErrorCollector) Errors() []*ValidationError {
	return c.errors
}

// Error returns the error message.
func (c *ValidationErrorCollector) Error() string {
	var errorMessages []string
	for _, err := range c.errors {
		errorMessages = append(errorMessages, err.Error())
	}
	return strings.Join(errorMessages, ", ")
}

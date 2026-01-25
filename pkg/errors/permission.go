package errors

import (
	"fmt"
	"strings"
)

// PermissionError is an error with a field and a list of messages.
type PermissionError struct {
	Code     int      `json:"code"`
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// NewPermissionError creates a new permission error.
func NewPermissionError(code int, field string, messages ...string) *PermissionError {
	return &PermissionError{
		Code:     code,
		Field:    field,
		Messages: messages,
	}
}

// Error returns the error message.
func (e *PermissionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, strings.Join(e.Messages, ", "))
}

// PermissionErrorCollector collects multiple permission errors.
type PermissionErrorCollector struct {
	errors []*PermissionError
}

// NewPermissionErrorCollector creates a new permission error collector.
func NewPermissionErrorCollector() *PermissionErrorCollector {
	return &PermissionErrorCollector{
		errors: make([]*PermissionError, 0),
	}
}

// Add adds a new permission error to the collector and returns the collector for chaining.
func (c *PermissionErrorCollector) Add(err *PermissionError) *PermissionErrorCollector {
	c.errors = append(c.errors, err)
	return c
}

// HasError returns true if the collector has any error.
func (c *PermissionErrorCollector) HasError() bool {
	return len(c.errors) > 0
}

// Errors returns the list of errors.
func (c *PermissionErrorCollector) Errors() []*PermissionError {
	return c.errors
}

// Error returns the error message.
func (c *PermissionErrorCollector) Error() string {
	var errorMessages []string
	for _, err := range c.errors {
		errorMessages = append(errorMessages, err.Error())
	}
	return strings.Join(errorMessages, ", ")
}

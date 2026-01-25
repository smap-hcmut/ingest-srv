package errors

import (
	"strings"
)

// PaymentError is an error with a field and a list of messages.
type PaymentError struct {
	Code     int      `json:"code"`
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

func NewPaymentError(code int, field string, messages ...string) *PaymentError {
	return &PaymentError{
		Code:     code,
		Field:    field,
		Messages: messages,
	}
}

// Error returns the error message.
func (e PaymentError) Error() string {
	return e.Field
}

type PaymentErrorCollector struct {
	errors []*PaymentError
}

func NewPaymentErrorCollector() *PaymentErrorCollector {
	return &PaymentErrorCollector{}
}

// Add adds a new Payment error to the collector.
func (c *PaymentErrorCollector) Add(err *PaymentError) PaymentErrorCollector {
	c.errors = append(c.errors, err)
	return *c
}

// HasError returns true if the collector has any error.
func (c PaymentErrorCollector) HasError() bool {
	return len(c.errors) > 0
}

// Errors returns the list of errors.
func (c PaymentErrorCollector) Errors() []*PaymentError {
	return c.errors
}

// Error returns the error message.
func (c PaymentErrorCollector) Error() string {
	var errors []string
	for _, err := range c.errors {
		errors = append(errors, err.Error())
	}
	return strings.Join(errors, ", ")
}

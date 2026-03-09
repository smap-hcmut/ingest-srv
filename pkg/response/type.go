package response

import (
	"encoding/json"
	"time"

	"ingest-srv/pkg/errors"
)

// Resp is the standard JSON response body.
type Resp struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Errors    any    `json:"errors,omitempty"`
}

// ErrorMapping maps errors to HTTPError for ErrorWithMap.
type ErrorMapping map[error]*errors.HTTPError

// Date is a date that marshals as DateFormat.
type Date time.Time

// MarshalJSON implements json.Marshaler for Date.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Local().Format(DateFormat))
}

// DateTime is a datetime that marshals as DateTimeFormat.
type DateTime time.Time

// MarshalJSON implements json.Marshaler for DateTime.
func (d DateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Local().Format(DateTimeFormat))
}

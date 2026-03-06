package errors

// ValidationError is an error with a field and messages.
type ValidationError struct {
	Code     int      `json:"code"`
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ValidationErrorCollector collects multiple validation errors.
type ValidationErrorCollector struct {
	errors []*ValidationError
}

// PermissionError is an error with a field and messages.
type PermissionError struct {
	Code     int      `json:"code"`
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// PermissionErrorCollector collects multiple permission errors.
type PermissionErrorCollector struct {
	errors []*PermissionError
}

// HTTPError represents an HTTP error with status code and message.
type HTTPError struct {
	Code       int
	Message    string
	StatusCode int
}

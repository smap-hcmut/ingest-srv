package results

import "errors"

var (
	// General errors
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrDuplicate    = errors.New("duplicate")
	ErrTemporary    = errors.New("temporary")
	ErrPermission   = errors.New("permission denied")

	// Content extraction errors
	ErrExtractContentFailed   = errors.New("failed to extract content")
	ErrMarshalPayloadFailed   = errors.New("failed to marshal payload to JSON")
	ErrUnmarshalContentFailed = errors.New("failed to unmarshal to CrawlerContent array")

	// Validation errors
	ErrValidationFailed      = errors.New("validation failed")
	ErrMissingRequiredFields = errors.New("missing required fields")

	// Mapping errors
	ErrMapContentFailed     = errors.New("failed to map content")
	ErrMapMetaFailed        = errors.New("failed to map meta")
	ErrMapContentDataFailed = errors.New("failed to map content data")
	ErrMapInteractionFailed = errors.New("failed to map interaction")
	ErrMapCommentsFailed    = errors.New("failed to map comments")

	// Timestamp errors
	ErrInvalidTimestamp           = errors.New("invalid timestamp")
	ErrEmptyTimestamp             = errors.New("empty timestamp")
	ErrUnsupportedTimestampFormat = errors.New("unsupported timestamp format")
)

package minio

import "fmt"

const (
	// ErrCodeConnection indicates a connection error occurred.
	ErrCodeConnection = "CONNECTION_ERROR"
	// ErrCodeBucketNotFound indicates the requested bucket does not exist.
	ErrCodeBucketNotFound = "BUCKET_NOT_FOUND"
	// ErrCodeObjectNotFound indicates the requested object does not exist.
	ErrCodeObjectNotFound = "OBJECT_NOT_FOUND"
	// ErrCodePermission indicates a permission denied error.
	ErrCodePermission = "PERMISSION_DENIED"
	// ErrCodeInvalidInput indicates invalid input parameters.
	ErrCodeInvalidInput = "INVALID_INPUT"
)

// StorageError represents an error that occurred during a MinIO storage operation.
type StorageError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation"`
	Cause     error  `json:"-"`
}

// Error returns the error message.
func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}
	return e.Message
}

// Unwrap returns the underlying error that caused this StorageError.
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewConnectionError creates a new StorageError for connection failures.
func NewConnectionError(err error) *StorageError {
	return &StorageError{Code: ErrCodeConnection, Message: "Storage connection failed", Cause: err}
}

// NewBucketNotFoundError creates a new StorageError for bucket not found errors.
func NewBucketNotFoundError(bucketName string) *StorageError {
	return &StorageError{Code: ErrCodeBucketNotFound, Message: "Bucket not found: " + bucketName}
}

// NewObjectNotFoundError creates a new StorageError for object not found errors.
func NewObjectNotFoundError(objectName string) *StorageError {
	return &StorageError{Code: ErrCodeObjectNotFound, Message: "Object not found: " + objectName}
}

// NewInvalidInputError creates a new StorageError for invalid input errors.
func NewInvalidInputError(message string) *StorageError {
	return &StorageError{Code: ErrCodeInvalidInput, Message: message}
}

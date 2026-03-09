package minio

import "time"

const (
	// HTTP transport for MinIO client
	maxIdleConns        = 100
	maxIdleConnsPerHost = 100
	idleConnTimeout     = 90 * time.Second
	disableCompression  = true
	disableKeepAlives   = false
)

const (
	// DefaultAsyncWorkers is the default number of async upload workers.
	DefaultAsyncWorkers = 4
	// DefaultAsyncQueueSize is the default async upload queue size.
	DefaultAsyncQueueSize = 100
	// DefaultListMaxKeys is the default max keys when listing objects.
	DefaultListMaxKeys = 1000
	// MaxListMaxKeys is the maximum allowed max keys for list.
	MaxListMaxKeys = 1000
	// MaxFileSizeBytes is the maximum upload file size (5GB).
	MaxFileSizeBytes = 5 * 1024 * 1024 * 1024
	// MaxPresignedExpiry is the maximum presigned URL expiry (7 days).
	MaxPresignedExpiry = 7 * 24 * time.Hour
	// DefaultEndpointPort is appended to endpoint if no port.
	DefaultEndpointPort = ":9000"
	// CleanupInterval is how often completed async tasks are cleaned up.
	CleanupInterval = 5 * time.Minute
	// CleanupMaxAge is the max age of completed tasks before cleanup.
	CleanupMaxAge = 1 * time.Hour
	// WaitForUploadPollInterval is the poll interval when waiting for upload.
	WaitForUploadPollInterval = 100 * time.Millisecond
)

// Content disposition values for download.
const (
	DispositionAuto       = "auto"
	DispositionInline     = "inline"
	DispositionAttachment = "attachment"
)

// Presigned URL methods.
const (
	MethodGET = "GET"
	MethodPUT = "PUT"
)

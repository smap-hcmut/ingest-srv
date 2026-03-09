package minio

import (
	"context"
	"io"
	"sync"
	"time"

	"ingest-srv/config"

	"github.com/minio/minio-go/v7"
)

// implMinIO implements MinIO.
type implMinIO struct {
	minioClient    *minio.Client
	config         *config.MinIOConfig
	mu             sync.RWMutex
	connected      bool
	asyncUploadMgr *asyncUploadManager
}

// FileInfo represents metadata about a file stored in MinIO.
type FileInfo struct {
	ID           string            `json:"id"`
	BucketName   string            `json:"bucket_name"`
	ObjectName   string            `json:"object_name"`
	OriginalName string            `json:"original_name"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	ETag         string            `json:"etag"`
	LastModified time.Time         `json:"last_modified"`
	Metadata     map[string]string `json:"metadata"`
	URL          string            `json:"url,omitempty"`
}

// UploadRequest contains the parameters for uploading a file to MinIO.
type UploadRequest struct {
	BucketName   string            `json:"bucket_name"`
	ObjectName   string            `json:"object_name"`
	OriginalName string            `json:"original_name"`
	Reader       io.Reader         `json:"-"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	Metadata     map[string]string `json:"metadata"`
}

// DownloadRequest contains the parameters for downloading a file from MinIO.
type DownloadRequest struct {
	BucketName  string     `json:"bucket_name"`
	ObjectName  string     `json:"object_name"`
	Range       *ByteRange `json:"range,omitempty"`
	Disposition string     `json:"disposition"` // "auto", "inline", "attachment"
}

// ByteRange represents a byte range for partial file downloads.
type ByteRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// ListRequest contains the parameters for listing files in a bucket.
type ListRequest struct {
	BucketName string `json:"bucket_name"`
	Prefix     string `json:"prefix"`
	Recursive  bool   `json:"recursive"`
	MaxKeys    int    `json:"max_keys"`
}

// ListResponse contains the result of listing files in a bucket.
type ListResponse struct {
	Files       []*FileInfo `json:"files"`
	IsTruncated bool        `json:"is_truncated"`
	NextMarker  string      `json:"next_marker,omitempty"`
	TotalCount  int         `json:"total_count"`
}

// PresignedURLRequest contains the parameters for generating a presigned URL.
type PresignedURLRequest struct {
	BucketName string            `json:"bucket_name"`
	ObjectName string            `json:"object_name"`
	Method     string            `json:"method"`
	Expiry     time.Duration     `json:"expiry"`
	Headers    map[string]string `json:"headers"`
}

// PresignedURLResponse contains the generated presigned URL and its metadata.
type PresignedURLResponse struct {
	URL       string            `json:"url"`
	ExpiresAt time.Time         `json:"expires_at"`
	Headers   map[string]string `json:"headers,omitempty"`
	Method    string            `json:"method"`
}

// DownloadHeaders contains HTTP headers for file download responses.
type DownloadHeaders struct {
	ContentType        string
	ContentDisposition string
	ContentLength      string
	LastModified       string
	ETag               string
	CacheControl       string
	AcceptRanges       string
	ContentRange       string
}

// BucketInfo contains information about a MinIO bucket.
type BucketInfo struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
	Region       string    `json:"region"`
}

// UploadStatus represents the status of an async upload.
type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploading UploadStatus = "uploading"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusFailed    UploadStatus = "failed"
	UploadStatusCancelled UploadStatus = "cancelled"
)

// AsyncUploadTask represents a single async upload task.
type AsyncUploadTask struct {
	ID           string
	Request      *UploadRequest
	ResultChan   chan *AsyncUploadResult
	ProgressChan chan *UploadProgress
	CreatedAt    time.Time
	ctx          context.Context
	cancel       context.CancelFunc
}

// AsyncUploadResult contains the result of an async upload.
type AsyncUploadResult struct {
	TaskID    string
	FileInfo  *FileInfo
	Error     error
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// UploadProgress represents the progress of an upload.
type UploadProgress struct {
	TaskID        string       `json:"task_id"`
	BytesUploaded int64        `json:"bytes_uploaded"`
	TotalBytes    int64        `json:"total_bytes"`
	Percentage    float64      `json:"percentage"`
	Status        UploadStatus `json:"status"`
	Error         string       `json:"error,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// asyncUploadManager manages async upload operations.
type asyncUploadManager struct {
	minio         *implMinIO
	workerPool    int
	uploadQueue   chan *AsyncUploadTask
	statusTracker *uploadStatusTracker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	started       bool
	mu            sync.RWMutex
}

// uploadStatusTracker tracks the status of async uploads.
type uploadStatusTracker struct {
	statuses map[string]*UploadProgress
	results  map[string]*AsyncUploadResult
	mu       sync.RWMutex
}

// progressReader wraps an io.Reader to track upload progress.
type progressReader struct {
	Reader     io.Reader
	TotalBytes int64
	bytesRead  int64
	OnProgress func(bytesRead int64)
	mu         sync.Mutex
}

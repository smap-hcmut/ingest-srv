package minio

import (
	"context"
	"io"
	"net/http"
	"time"

	"ingest-srv/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO is the composite interface embedding all sub-interfaces.
type MinIO interface {
	Connection
	BucketManager
	FileUploader
	FileDownloader
	FileManager
	FileLister
	MetadataManager
	AsyncUploader
}

// Connection defines interface for MinIO connection operations.
type Connection interface {
	Connect(ctx context.Context) error
	ConnectWithRetry(ctx context.Context, maxRetries int) error
	HealthCheck(ctx context.Context) error
	Close() error
}

// BucketManager defines operations for managing buckets.
type BucketManager interface {
	CreateBucket(ctx context.Context, bucketName string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	ListBuckets(ctx context.Context) ([]*BucketInfo, error)
}

// FileUploader defines methods for uploading files.
type FileUploader interface {
	UploadFile(ctx context.Context, req *UploadRequest) (*FileInfo, error)
	GetPresignedUploadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error)
}

// FileDownloader defines methods for downloading files.
type FileDownloader interface {
	DownloadFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error)
	StreamFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error)
	GetPresignedDownloadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error)
}

// FileManager defines methods for file metadata, manipulation, and existence checks.
type FileManager interface {
	GetFileInfo(ctx context.Context, bucketName, objectName string) (*FileInfo, error)
	DeleteFile(ctx context.Context, bucketName, objectName string) error
	CopyFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error
	MoveFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error
	FileExists(ctx context.Context, bucketName, objectName string) (bool, error)
}

// FileLister provides file/bucket listing.
type FileLister interface {
	ListFiles(ctx context.Context, req *ListRequest) (*ListResponse, error)
}

// MetadataManager for metadata operations.
type MetadataManager interface {
	UpdateMetadata(ctx context.Context, bucketName, objectName string, metadata map[string]string) error
	GetMetadata(ctx context.Context, bucketName, objectName string) (map[string]string, error)
}

// AsyncUploader for async file uploads.
type AsyncUploader interface {
	UploadAsync(ctx context.Context, req *UploadRequest) (taskID string, err error)
	GetUploadStatus(taskID string) (*UploadProgress, error)
	WaitForUpload(taskID string, timeout time.Duration) (*AsyncUploadResult, error)
	CancelUpload(taskID string) error
}

// NewMinIO creates a new MinIO client. Returns the MinIO interface.
func NewMinIO(cfg *config.MinIOConfig) (MinIO, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		DisableCompression:  disableCompression,
		DisableKeepAlives:   disableKeepAlives,
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:    cfg.UseSSL,
		Region:    cfg.Region,
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}

	impl := &implMinIO{
		minioClient: client,
		config:      cfg,
		connected:   false,
	}

	workerPoolSize := DefaultAsyncWorkers
	queueSize := DefaultAsyncQueueSize
	if cfg.AsyncUploadWorkers > 0 {
		workerPoolSize = cfg.AsyncUploadWorkers
	}
	if cfg.AsyncUploadQueueSize > 0 {
		queueSize = cfg.AsyncUploadQueueSize
	}

	impl.asyncUploadMgr = newAsyncUploadManager(impl, workerPoolSize, queueSize)
	impl.asyncUploadMgr.start()

	return impl, nil
}

// NewMinIOWithRetry creates a new MinIO client and connects with retry.
func NewMinIOWithRetry(cfg *config.MinIOConfig, maxRetries int) (MinIO, error) {
	client, err := NewMinIO(cfg)
	if err != nil {
		return nil, err
	}
	if err := client.ConnectWithRetry(context.Background(), maxRetries); err != nil {
		return nil, err
	}
	return client, nil
}

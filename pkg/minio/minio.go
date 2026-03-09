package minio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// --- implMinIO: connection ---

func (m *implMinIO) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.minioClient.ListBuckets(ctx)
	if err != nil {
		m.connected = false
		return handleMinIOError(err, "connect")
	}
	m.connected = true
	return nil
}

func (m *implMinIO) ConnectWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := m.Connect(ctx); err == nil {
			return nil
		} else {
			lastErr = err
			backoff := time.Duration(1<<uint(i)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}
	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, lastErr)
}

func (m *implMinIO) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.connected {
		return NewConnectionError(fmt.Errorf("not connected"))
	}
	_, err := m.minioClient.ListBuckets(ctx)
	if err != nil {
		return handleMinIOError(err, "health_check")
	}
	return nil
}

func (m *implMinIO) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// --- implMinIO: bucket ---

func (m *implMinIO) CreateBucket(ctx context.Context, bucketName string) error {
	if err := validateBucketName(bucketName); err != nil {
		return err
	}
	exists, err := m.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return handleMinIOError(err, "check_bucket_exists")
	}
	if exists {
		return NewInvalidInputError(fmt.Sprintf("bucket already exists: %s", bucketName))
	}
	err = m.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: m.config.Region})
	if err != nil {
		return handleMinIOError(err, "create_bucket")
	}
	return nil
}

func (m *implMinIO) DeleteBucket(ctx context.Context, bucketName string) error {
	if err := validateBucketName(bucketName); err != nil {
		return err
	}
	return handleMinIOError(m.minioClient.RemoveBucket(ctx, bucketName), "delete_bucket")
}

func (m *implMinIO) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if err := validateBucketName(bucketName); err != nil {
		return false, err
	}
	exists, err := m.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return false, handleMinIOError(err, "check_bucket_exists")
	}
	return exists, nil
}

func (m *implMinIO) ListBuckets(ctx context.Context) ([]*BucketInfo, error) {
	buckets, err := m.minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, handleMinIOError(err, "list_buckets")
	}
	var result []*BucketInfo
	for _, bucket := range buckets {
		result = append(result, &BucketInfo{
			Name:         bucket.Name,
			CreationDate: bucket.CreationDate,
			Region:       m.config.Region,
		})
	}
	return result, nil
}

// --- implMinIO: upload / download ---

func (m *implMinIO) UploadFile(ctx context.Context, req *UploadRequest) (*FileInfo, error) {
	if err := validateUploadRequest(req); err != nil {
		return nil, err
	}
	opts := minio.PutObjectOptions{ContentType: req.ContentType}
	if req.Metadata != nil {
		opts.UserMetadata = req.Metadata
	} else {
		opts.UserMetadata = make(map[string]string)
	}
	if req.OriginalName != "" {
		opts.UserMetadata["original-name"] = req.OriginalName
	}
	info, err := m.minioClient.PutObject(ctx, req.BucketName, req.ObjectName, req.Reader, req.Size, opts)
	if err != nil {
		return nil, handleMinIOError(err, "upload_file")
	}
	return &FileInfo{
		BucketName:   req.BucketName,
		ObjectName:   req.ObjectName,
		OriginalName: req.OriginalName,
		Size:         info.Size,
		ContentType:  req.ContentType,
		ETag:         info.ETag,
		LastModified: time.Now(),
		Metadata:     req.Metadata,
	}, nil
}

func (m *implMinIO) GetPresignedUploadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error) {
	if err := validatePresignedURLRequest(req); err != nil {
		return nil, err
	}
	url, err := m.minioClient.PresignedPutObject(ctx, req.BucketName, req.ObjectName, req.Expiry)
	if err != nil {
		return nil, handleMinIOError(err, "get_presigned_upload_url")
	}
	return &PresignedURLResponse{
		URL:       url.String(),
		ExpiresAt: time.Now().Add(req.Expiry),
		Method:    MethodPUT,
		Headers:   req.Headers,
	}, nil
}

func (m *implMinIO) DownloadFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error) {
	if err := validateDownloadRequest(req); err != nil {
		return nil, nil, err
	}
	objInfo, err := m.minioClient.StatObject(ctx, req.BucketName, req.ObjectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, handleMinIOError(err, "get_file_info")
	}
	opts := minio.GetObjectOptions{}
	if req.Range != nil {
		opts.SetRange(req.Range.Start, req.Range.End)
	}
	object, err := m.minioClient.GetObject(ctx, req.BucketName, req.ObjectName, opts)
	if err != nil {
		return nil, nil, handleMinIOError(err, "download_file")
	}
	return object, m.generateDownloadHeaders(objInfo, req), nil
}

func (m *implMinIO) StreamFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error) {
	req.Disposition = DispositionInline
	reader, headers, err := m.DownloadFile(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	headers.CacheControl = "public, max-age=86400"
	headers.AcceptRanges = "bytes"
	if req.Range != nil {
		headers.ContentRange = fmt.Sprintf("bytes %d-%d/%s", req.Range.Start, req.Range.End, headers.ContentLength)
	}
	return reader, headers, nil
}

func (m *implMinIO) GetPresignedDownloadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error) {
	if err := validatePresignedURLRequest(req); err != nil {
		return nil, err
	}
	url, err := m.minioClient.PresignedGetObject(ctx, req.BucketName, req.ObjectName, req.Expiry, nil)
	if err != nil {
		return nil, handleMinIOError(err, "get_presigned_download_url")
	}
	return &PresignedURLResponse{
		URL:       url.String(),
		ExpiresAt: time.Now().Add(req.Expiry),
		Method:    MethodGET,
		Headers:   req.Headers,
	}, nil
}

func (m *implMinIO) generateDownloadHeaders(objInfo minio.ObjectInfo, req *DownloadRequest) *DownloadHeaders {
	disposition := m.determineContentDisposition(objInfo.ContentType, req.Disposition)
	originalName := objInfo.UserMetadata["original-name"]
	if originalName == "" {
		originalName = objInfo.Key
	}
	headers := &DownloadHeaders{
		ContentType:        objInfo.ContentType,
		ContentDisposition: fmt.Sprintf("%s; filename=\"%s\"", disposition, originalName),
		ContentLength:      fmt.Sprintf("%d", objInfo.Size),
		LastModified:       objInfo.LastModified.Format(http.TimeFormat),
		ETag:               objInfo.ETag,
		AcceptRanges:       "bytes",
	}
	if disposition == DispositionInline {
		headers.CacheControl = "public, max-age=3600"
	} else {
		headers.CacheControl = "private, no-cache"
	}
	return headers
}

func (m *implMinIO) determineContentDisposition(contentType, requestedDisposition string) string {
	if requestedDisposition == DispositionInline || requestedDisposition == DispositionAttachment {
		return requestedDisposition
	}
	if requestedDisposition == DispositionAuto {
		viewableTypes := []string{"image/", "video/", "audio/", "application/pdf", "text/plain", "text/html", "application/json", "application/xml"}
		for _, viewable := range viewableTypes {
			if strings.HasPrefix(contentType, viewable) {
				return DispositionInline
			}
		}
		return DispositionAttachment
	}
	return DispositionAttachment
}

// --- implMinIO: file info / delete / copy / move / exists ---

func (m *implMinIO) GetFileInfo(ctx context.Context, bucketName, objectName string) (*FileInfo, error) {
	if err := validateBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := validateObjectName(objectName); err != nil {
		return nil, err
	}
	objInfo, err := m.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, handleMinIOError(err, "get_file_info")
	}
	fileInfo := &FileInfo{
		BucketName:   bucketName,
		ObjectName:   objectName,
		Size:         objInfo.Size,
		ContentType:  objInfo.ContentType,
		ETag:         objInfo.ETag,
		LastModified: objInfo.LastModified,
		Metadata:     objInfo.UserMetadata,
	}
	if originalName, exists := objInfo.UserMetadata["original-name"]; exists {
		fileInfo.OriginalName = originalName
	}
	return fileInfo, nil
}

func (m *implMinIO) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	if err := validateBucketName(bucketName); err != nil {
		return err
	}
	if err := validateObjectName(objectName); err != nil {
		return err
	}
	return handleMinIOError(m.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{}), "delete_file")
}

func (m *implMinIO) CopyFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error {
	_, err := m.minioClient.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: destBucket, Object: destObject},
		minio.CopySrcOptions{Bucket: srcBucket, Object: srcObject})
	return handleMinIOError(err, "copy_file")
}

func (m *implMinIO) MoveFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error {
	if err := m.CopyFile(ctx, srcBucket, srcObject, destBucket, destObject); err != nil {
		return err
	}
	if err := m.DeleteFile(ctx, srcBucket, srcObject); err != nil {
		if cleanupErr := m.DeleteFile(ctx, destBucket, destObject); cleanupErr != nil {
			return fmt.Errorf("move failed: %w, cleanup also failed: %v", err, cleanupErr)
		}
		return fmt.Errorf("move failed: %w", err)
	}
	return nil
}

func (m *implMinIO) FileExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := m.GetFileInfo(ctx, bucketName, objectName)
	if err != nil {
		if storageErr, ok := err.(*StorageError); ok && storageErr.Code == ErrCodeObjectNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// --- implMinIO: list / metadata ---

func (m *implMinIO) ListFiles(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	if err := validateListRequest(req); err != nil {
		return nil, err
	}
	opts := minio.ListObjectsOptions{Prefix: req.Prefix, Recursive: req.Recursive, MaxKeys: req.MaxKeys}
	var files []*FileInfo
	objectCh := m.minioClient.ListObjects(ctx, req.BucketName, opts)
	for object := range objectCh {
		if object.Err != nil {
			return nil, handleMinIOError(object.Err, "list_files")
		}
		files = append(files, &FileInfo{
			BucketName:   req.BucketName,
			ObjectName:   object.Key,
			Size:         object.Size,
			ETag:         object.ETag,
			LastModified: object.LastModified,
			ContentType:  object.ContentType,
		})
	}
	resp := &ListResponse{Files: files, TotalCount: len(files), IsTruncated: len(files) >= req.MaxKeys}
	if resp.IsTruncated && len(files) > 0 {
		resp.NextMarker = files[len(files)-1].ObjectName
	}
	return resp, nil
}

func (m *implMinIO) UpdateMetadata(ctx context.Context, bucketName, objectName string, metadata map[string]string) error {
	_, err := m.minioClient.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: bucketName, Object: objectName, UserMetadata: metadata, ReplaceMetadata: true},
		minio.CopySrcOptions{Bucket: bucketName, Object: objectName})
	return handleMinIOError(err, "update_metadata")
}

func (m *implMinIO) GetMetadata(ctx context.Context, bucketName, objectName string) (map[string]string, error) {
	fileInfo, err := m.GetFileInfo(ctx, bucketName, objectName)
	if err != nil {
		return nil, err
	}
	return fileInfo.Metadata, nil
}

// --- implMinIO: async upload (delegate to manager) ---

func (m *implMinIO) UploadAsync(ctx context.Context, req *UploadRequest) (string, error) {
	return m.asyncUploadMgr.uploadAsync(ctx, req)
}

func (m *implMinIO) GetUploadStatus(taskID string) (*UploadProgress, error) {
	return m.asyncUploadMgr.getUploadStatus(taskID)
}

func (m *implMinIO) WaitForUpload(taskID string, timeout time.Duration) (*AsyncUploadResult, error) {
	return m.asyncUploadMgr.waitForUpload(taskID, timeout)
}

func (m *implMinIO) CancelUpload(taskID string) error {
	return m.asyncUploadMgr.cancelUpload(taskID)
}

// --- helpers ---

func handleMinIOError(err error, operation string) *StorageError {
	if err == nil {
		return nil
	}
	if minioErr, ok := err.(minio.ErrorResponse); ok {
		switch minioErr.Code {
		case "NoSuchBucket":
			return NewBucketNotFoundError("")
		case "NoSuchKey":
			return NewObjectNotFoundError("")
		case "AccessDenied":
			return &StorageError{Code: ErrCodePermission, Message: "Access denied", Operation: operation, Cause: err}
		default:
			return &StorageError{Code: ErrCodeConnection, Message: fmt.Sprintf("MinIO operation failed: %s", minioErr.Code), Operation: operation, Cause: err}
		}
	}
	return NewConnectionError(err)
}

// --- async upload manager ---

func newUploadStatusTracker() *uploadStatusTracker {
	return &uploadStatusTracker{
		statuses: make(map[string]*UploadProgress),
		results:  make(map[string]*AsyncUploadResult),
	}
}

func newAsyncUploadManager(impl *implMinIO, workerPoolSize, queueSize int) *asyncUploadManager {
	if workerPoolSize <= 0 {
		workerPoolSize = DefaultAsyncWorkers
	}
	if queueSize <= 0 {
		queueSize = DefaultAsyncQueueSize
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &asyncUploadManager{
		minio:         impl,
		workerPool:    workerPoolSize,
		uploadQueue:   make(chan *AsyncUploadTask, queueSize),
		statusTracker: newUploadStatusTracker(),
		ctx:           ctx,
		cancel:        cancel,
		started:       false,
	}
}

func (m *asyncUploadManager) start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return
	}
	for i := 0; i < m.workerPool; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
	m.wg.Add(1)
	go m.cleanupWorker()
	m.started = true
}

func (m *asyncUploadManager) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return
	}
	m.cancel()
	close(m.uploadQueue)
	m.wg.Wait()
	m.started = false
}

func (m *asyncUploadManager) uploadAsync(ctx context.Context, req *UploadRequest) (string, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return "", fmt.Errorf("async upload manager not started")
	}
	m.mu.RUnlock()

	taskID := uuid.New().String()
	taskCtx, taskCancel := context.WithCancel(ctx)
	task := &AsyncUploadTask{
		ID:           taskID,
		Request:      req,
		ResultChan:   make(chan *AsyncUploadResult, 1),
		ProgressChan: make(chan *UploadProgress, 10),
		CreatedAt:    time.Now(),
		ctx:          taskCtx,
		cancel:       taskCancel,
	}
	m.statusTracker.updateStatus(taskID, &UploadProgress{
		TaskID: taskID, TotalBytes: req.Size, Status: UploadStatusPending, UpdatedAt: time.Now(),
	})
	select {
	case m.uploadQueue <- task:
		return taskID, nil
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "", fmt.Errorf("upload queue is full")
	}
}

func (m *asyncUploadManager) getUploadStatus(taskID string) (*UploadProgress, error) {
	progress, exists := m.statusTracker.getStatus(taskID)
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return progress, nil
}

func (m *asyncUploadManager) waitForUpload(taskID string, timeout time.Duration) (*AsyncUploadResult, error) {
	progress, exists := m.statusTracker.getStatus(taskID)
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if progress.Status == UploadStatusCompleted || progress.Status == UploadStatusFailed {
		result := m.statusTracker.getResult(taskID)
		if result != nil {
			return result, nil
		}
		return nil, fmt.Errorf("task %s is %s but result not available", taskID, progress.Status)
	}
	ticker := time.NewTicker(WaitForUploadPollInterval)
	defer ticker.Stop()
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout waiting for upload: %s", taskID)
		case <-ticker.C:
			progress, exists = m.statusTracker.getStatus(taskID)
			if !exists {
				return nil, fmt.Errorf("task disappeared: %s", taskID)
			}
			if progress.Status == UploadStatusCompleted || progress.Status == UploadStatusFailed {
				result := m.statusTracker.getResult(taskID)
				if result != nil {
					return result, nil
				}
				return nil, fmt.Errorf("task %s is %s but result not available", taskID, progress.Status)
			}
		}
	}
}

func (m *asyncUploadManager) cancelUpload(taskID string) error {
	progress, exists := m.statusTracker.getStatus(taskID)
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if progress.Status != UploadStatusPending && progress.Status != UploadStatusUploading {
		return fmt.Errorf("cannot cancel task in status: %s", progress.Status)
	}
	m.statusTracker.updateStatus(taskID, &UploadProgress{
		TaskID: taskID, Status: UploadStatusCancelled, UpdatedAt: time.Now(),
	})
	return nil
}

func (m *asyncUploadManager) worker(workerID int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.ctx.Done():
			return
		case task, ok := <-m.uploadQueue:
			if !ok {
				return
			}
			m.processUploadTask(workerID, task)
		}
	}
}

func (m *asyncUploadManager) processUploadTask(workerID int, task *AsyncUploadTask) {
	startTime := time.Now()
	progress, _ := m.statusTracker.getStatus(task.ID)
	if progress != nil && progress.Status == UploadStatusCancelled {
		return
	}
	m.statusTracker.updateStatus(task.ID, &UploadProgress{
		TaskID: task.ID, TotalBytes: task.Request.Size, Status: UploadStatusUploading, UpdatedAt: time.Now(),
	})
	pr := &progressReader{
		Reader:     task.Request.Reader,
		TotalBytes: task.Request.Size,
		OnProgress: func(bytesRead int64) {
			progress, _ := m.statusTracker.getStatus(task.ID)
			if progress != nil && progress.Status == UploadStatusCancelled {
				return
			}
			percentage := float64(bytesRead) / float64(task.Request.Size) * 100
			if percentage > 100 {
				percentage = 100
			}
			m.statusTracker.updateStatus(task.ID, &UploadProgress{
				TaskID: task.ID, BytesUploaded: bytesRead, TotalBytes: task.Request.Size,
				Percentage: percentage, Status: UploadStatusUploading, UpdatedAt: time.Now(),
			})
		},
	}
	task.Request.Reader = pr
	fileInfo, err := m.minio.UploadFile(task.ctx, task.Request)
	duration := time.Since(startTime)
	endTime := time.Now()
	result := &AsyncUploadResult{
		TaskID: task.ID, FileInfo: fileInfo, Error: err, Duration: duration, StartTime: startTime, EndTime: endTime,
	}
	if err != nil {
		m.statusTracker.updateStatus(task.ID, &UploadProgress{
			TaskID: task.ID, Status: UploadStatusFailed, Error: err.Error(), UpdatedAt: time.Now(),
		})
	} else {
		m.statusTracker.updateStatus(task.ID, &UploadProgress{
			TaskID: task.ID, BytesUploaded: task.Request.Size, TotalBytes: task.Request.Size,
			Percentage: 100, Status: UploadStatusCompleted, UpdatedAt: time.Now(),
		})
	}
	m.statusTracker.storeResult(task.ID, result)
	select {
	case task.ResultChan <- result:
	default:
	}
}

func (m *asyncUploadManager) cleanupWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.statusTracker.cleanupOldStatuses(CleanupMaxAge)
		}
	}
}

// --- upload status tracker ---

func (t *uploadStatusTracker) updateStatus(taskID string, progress *UploadProgress) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, exists := t.statuses[taskID]; exists {
		if progress.BytesUploaded > 0 {
			existing.BytesUploaded = progress.BytesUploaded
		}
		if progress.TotalBytes > 0 {
			existing.TotalBytes = progress.TotalBytes
		}
		if progress.Percentage > 0 {
			existing.Percentage = progress.Percentage
		}
		if progress.Status != "" {
			existing.Status = progress.Status
		}
		if progress.Error != "" {
			existing.Error = progress.Error
		}
		existing.UpdatedAt = progress.UpdatedAt
	} else {
		t.statuses[taskID] = progress
	}
}

func (t *uploadStatusTracker) getStatus(taskID string) (*UploadProgress, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	progress, exists := t.statuses[taskID]
	if !exists {
		return nil, false
	}
	progressCopy := *progress
	return &progressCopy, true
}

func (t *uploadStatusTracker) storeResult(taskID string, result *AsyncUploadResult) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.results[taskID] = result
}

func (t *uploadStatusTracker) getResult(taskID string) *AsyncUploadResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.results[taskID]
}

func (t *uploadStatusTracker) cleanupOldStatuses(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	for taskID, progress := range t.statuses {
		if progress.Status == UploadStatusCompleted || progress.Status == UploadStatusFailed || progress.Status == UploadStatusCancelled {
			if now.Sub(progress.UpdatedAt) > maxAge {
				delete(t.statuses, taskID)
				delete(t.results, taskID)
			}
		}
	}
}

// --- progress reader ---

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.mu.Lock()
		pr.bytesRead += int64(n)
		bytesRead := pr.bytesRead
		pr.mu.Unlock()
		if pr.OnProgress != nil {
			pr.OnProgress(bytesRead)
		}
	}
	return n, err
}

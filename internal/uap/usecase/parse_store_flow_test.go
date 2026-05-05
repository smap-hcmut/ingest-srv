package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"

	sharedLog "github.com/smap-hcmut/shared-libs/go/log"
	sharedMinio "github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		mock        func(*uap.MockPublisher)
		outputTopic string
	}{
		"with publisher": {
			mock: func(p *uap.MockPublisher) {
				p.EXPECT().Topic().Return("uap-topic").Once()
			},
			outputTopic: "uap-topic",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			publisher := uap.NewMockPublisher(t)
			tc.mock(publisher)

			got := New(nil, nil, nil, "parsed-bucket", publisher)
			impl, ok := got.(*implUseCase)
			require.True(t, ok)
			require.Equal(t, "parsed-bucket", impl.outputBucket)
			require.Equal(t, tc.outputTopic, impl.publishTopic)
			require.NotNil(t, impl.parsers)
			require.NotNil(t, impl.subtitleHTTPClient)
			require.NotNil(t, impl.now)
		})
	}

	t.Run("without publisher", func(t *testing.T) {
		got := New(nil, nil, nil, "", nil)
		impl, ok := got.(*implUseCase)
		require.True(t, ok)
		require.Empty(t, impl.publishTopic)
		require.NotNil(t, impl.parsers)
	})
}

func TestParseAndStoreRawBatch(t *testing.T) {
	type mockSetup struct {
		repo      func(*repo.MockRepository)
		minio     *fakeUAPMinIO
		publisher func(*uap.MockPublisher)
	}

	baseInput := func() uap.ParseAndStoreRawBatchInput {
		return uap.ParseAndStoreRawBatchInput{
			RawBatchID:     "raw-batch-1",
			ProjectID:      "project-1",
			SourceID:       "source-1",
			TaskID:         "task-1",
			Platform:       "youtube",
			Action:         "full_flow",
			StorageBucket:  "raw-bucket",
			StoragePath:    "raw/path.json",
			BatchID:        "batch-1",
			RawMetadata:    json.RawMessage(`{"source":"crawler"}`),
			RequestPayload: json.RawMessage(`{"params":{"keyword":"bia"}}`),
			CompletionTime: time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC),
		}
	}

	rawBody := `{"result":{"videos":[{"video":{"video_id":"yt-1","title":"Video title","url":"https://www.youtube.com/watch?v=yt-1"}}]}}`
	errRepo := errors.New("repo failed")
	errDownload := errors.New("download failed")
	errRead := errors.New("read failed")
	errUpload := errors.New("upload failed")

	tcs := map[string]struct {
		input uap.ParseAndStoreRawBatchInput
		mock  mockSetup
		err   error
	}{
		"invalid input": {
			input: uap.ParseAndStoreRawBatchInput{},
			err:   uap.ErrInvalidRawBatchInput,
		},
		"unsupported parser": {
			input: func() uap.ParseAndStoreRawBatchInput {
				input := baseInput()
				input.Platform = "unknown"
				return input
			}(),
			err: uap.ErrInvalidRawBatchInput,
		},
		"claim error": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(false, errRepo).Once()
				},
			},
			err: errRepo,
		},
		"already claimed": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(false, nil).Once()
				},
			},
		},
		"download error marks failed": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && strings.Contains(opt.ErrorMessage, "download raw batch object")
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadErr: errDownload},
			},
			err: errDownload,
		},
		"read error marks failed": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && strings.Contains(opt.ErrorMessage, "read raw batch object")
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadReader: errReadCloser{err: errRead}},
			},
			err: errRead,
		},
		"mark downloaded error marks failed": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchDownloaded(mock.Anything, repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"}).Return(errRepo).Once()
					r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && strings.Contains(opt.ErrorMessage, "mark raw batch downloaded")
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadBody: rawBody},
			},
			err: errRepo,
		},
		"parser error marks failed": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchDownloaded(mock.Anything, repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"}).Return(nil).Once()
					r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && strings.Contains(opt.ErrorMessage, "parse raw batch")
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadBody: `{"result":`},
			},
			err: uap.ErrParseRawPayload,
		},
		"upload error marks failed with publish metadata": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchDownloaded(mock.Anything, repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"}).Return(nil).Once()
					r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && strings.Contains(opt.PublishError, "upload uap chunk")
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadBody: rawBody, uploadErr: errUpload},
				publisher: func(p *uap.MockPublisher) {
					p.EXPECT().Topic().Return("uap-topic").Once()
					p.EXPECT().Publish(mock.Anything, mock.MatchedBy(func(input uap.PublishUAPInput) bool {
						return input.Record.Identity.UAPID == "yt_p_yt-1"
					})).Return(nil).Once()
				},
			},
			err: errUpload,
		},
		"mark parsed error marks failed": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchDownloaded(mock.Anything, repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"}).Return(nil).Once()
					r.EXPECT().MarkRawBatchParsed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchParsedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && opt.PublishRecordCount == 1
					})).Return(errRepo).Once()
					r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && strings.Contains(opt.ErrorMessage, "mark raw batch parsed")
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadBody: rawBody},
			},
			err: errRepo,
		},
		"success": {
			input: baseInput(),
			mock: mockSetup{
				repo: func(r *repo.MockRepository) {
					r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
					r.EXPECT().MarkRawBatchDownloaded(mock.Anything, repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"}).Return(nil).Once()
					r.EXPECT().MarkRawBatchParsed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchParsedOptions) bool {
						return opt.RawBatchID == "raw-batch-1" && opt.PublishRecordCount == 1 && strings.Contains(string(opt.RawMetadata), uap.ArtifactsMetadataKey)
					})).Return(nil).Once()
				},
				minio: &fakeUAPMinIO{downloadBody: rawBody},
				publisher: func(p *uap.MockPublisher) {
					p.EXPECT().Topic().Return("uap-topic").Once()
					p.EXPECT().Publish(mock.Anything, mock.AnythingOfType("uap.PublishUAPInput")).Return(nil).Once()
				},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r := repo.NewMockRepository(t)
			if tc.mock.repo != nil {
				tc.mock.repo(r)
			}

			minioClient := tc.mock.minio
			if minioClient == nil {
				minioClient = &fakeUAPMinIO{}
			}

			var publisher uap.Publisher
			if tc.mock.publisher != nil {
				p := uap.NewMockPublisher(t)
				tc.mock.publisher(p)
				publisher = p
			}

			uc := New(noopUAPLogger{}, r, minioClient, "parsed-bucket", publisher)
			err := uc.ParseAndStoreRawBatch(context.Background(), tc.input)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPublishRecord(t *testing.T) {
	record := uap.UAPRecord{Identity: uap.UAPIdentity{UAPID: "uap-1"}}
	input := uap.ParseAndStoreRawBatchInput{RawBatchID: "raw-batch-1"}
	errPublish := errors.New("publish failed")

	tcs := map[string]struct {
		mock   func(*uap.MockPublisher)
		stats  *uap.KafkaPublishStats
		output uap.KafkaPublishStats
	}{
		"nil stats": {
			stats: nil,
		},
		"nil publisher": {
			stats:  &uap.KafkaPublishStats{},
			output: uap.KafkaPublishStats{},
		},
		"success": {
			stats: &uap.KafkaPublishStats{},
			mock: func(p *uap.MockPublisher) {
				p.EXPECT().Publish(mock.Anything, uap.PublishUAPInput{Record: record}).Return(nil).Once()
			},
			output: uap.KafkaPublishStats{AttemptedCount: 1, SuccessCount: 1},
		},
		"publish error": {
			stats: &uap.KafkaPublishStats{},
			mock: func(p *uap.MockPublisher) {
				p.EXPECT().Publish(mock.Anything, uap.PublishUAPInput{Record: record}).Return(errPublish).Once()
			},
			output: uap.KafkaPublishStats{AttemptedCount: 1, FailedCount: 1, LastError: errPublish.Error()},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := &implUseCase{l: noopUAPLogger{}}
			if tc.mock != nil {
				publisher := uap.NewMockPublisher(t)
				tc.mock(publisher)
				uc.publisher = publisher
			}

			uc.publishRecord(context.Background(), record, input, tc.stats)
			if tc.stats != nil {
				require.Equal(t, tc.output, *tc.stats)
			}
		})
	}
}

func TestStorageHelpers(t *testing.T) {
	uc := &implUseCase{repo: repo.NewMockRepository(t), minio: &fakeUAPMinIO{}}
	record := uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     "uap-1",
			OriginID:  "origin-1",
			UAPType:   uap.UAPTypePost,
			Platform:  "youtube",
			ProjectID: "project-1",
			TaskID:    "task-1",
		},
	}

	t.Run("validate input", func(t *testing.T) {
		valid := uap.ParseAndStoreRawBatchInput{
			RawBatchID:    "raw",
			ProjectID:     "project",
			SourceID:      "source",
			TaskID:        "task",
			StorageBucket: "bucket",
			StoragePath:   "path",
			BatchID:       "batch",
		}
		require.NoError(t, uc.validateParseAndStoreRawBatchInputCommon(valid))
		valid.BatchID = ""
		require.ErrorIs(t, uc.validateParseAndStoreRawBatchInputCommon(valid), uap.ErrInvalidRawBatchInput)
	})

	t.Run("upload chunk", func(t *testing.T) {
		client := &fakeUAPMinIO{}
		part, err := uc.uploadChunk(context.Background(), client, "bucket", " project ", " source ", " batch ", 3, []uap.UAPRecord{record})
		require.NoError(t, err)
		require.Equal(t, 3, part.PartNo)
		require.Equal(t, "bucket", part.StorageBucket)
		require.Equal(t, "uap-batches/project/source/batch/part-00003.jsonl", part.StoragePath)
		require.Equal(t, 1, part.RecordCount)
		require.Len(t, client.uploads, 1)
		require.Equal(t, uap.ContentTypeNDJSON, client.uploads[0].ContentType)
	})

	t.Run("read all and close", func(t *testing.T) {
		got, err := uc.readAllAndClose(io.NopCloser(strings.NewReader("body")))
		require.NoError(t, err)
		require.Equal(t, []byte("body"), got)

		_, err = uc.readAllAndClose(errReadCloser{err: errors.New("read failed")})
		require.Error(t, err)
	})
}

type fakeUAPMinIO struct {
	downloadBody   string
	downloadReader io.ReadCloser
	downloadErr    error
	uploadErr      error
	uploads        []*sharedMinio.UploadRequest
}

func (f *fakeUAPMinIO) Connect(context.Context) error { return nil }
func (f *fakeUAPMinIO) ConnectWithRetry(context.Context, int) error {
	return nil
}
func (f *fakeUAPMinIO) HealthCheck(context.Context) error { return nil }
func (f *fakeUAPMinIO) Close() error                      { return nil }
func (f *fakeUAPMinIO) CreateBucket(context.Context, string) error {
	return nil
}
func (f *fakeUAPMinIO) DeleteBucket(context.Context, string) error { return nil }
func (f *fakeUAPMinIO) BucketExists(context.Context, string) (bool, error) {
	return true, nil
}
func (f *fakeUAPMinIO) ListBuckets(context.Context) ([]*sharedMinio.BucketInfo, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) UploadFile(_ context.Context, req *sharedMinio.UploadRequest) (*sharedMinio.FileInfo, error) {
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	f.uploads = append(f.uploads, req)
	return &sharedMinio.FileInfo{BucketName: req.BucketName, ObjectName: req.ObjectName, Size: req.Size}, nil
}
func (f *fakeUAPMinIO) GetPresignedUploadURL(context.Context, *sharedMinio.PresignedURLRequest) (*sharedMinio.PresignedURLResponse, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) DownloadFile(context.Context, *sharedMinio.DownloadRequest) (io.ReadCloser, *sharedMinio.DownloadHeaders, error) {
	if f.downloadErr != nil {
		return nil, nil, f.downloadErr
	}
	if f.downloadReader != nil {
		return f.downloadReader, nil, nil
	}
	return io.NopCloser(strings.NewReader(f.downloadBody)), nil, nil
}
func (f *fakeUAPMinIO) StreamFile(ctx context.Context, req *sharedMinio.DownloadRequest) (io.ReadCloser, *sharedMinio.DownloadHeaders, error) {
	return f.DownloadFile(ctx, req)
}
func (f *fakeUAPMinIO) GetPresignedDownloadURL(context.Context, *sharedMinio.PresignedURLRequest) (*sharedMinio.PresignedURLResponse, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) GetFileInfo(context.Context, string, string) (*sharedMinio.FileInfo, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) DeleteFile(context.Context, string, string) error { return nil }
func (f *fakeUAPMinIO) CopyFile(context.Context, string, string, string, string) error {
	return nil
}
func (f *fakeUAPMinIO) MoveFile(context.Context, string, string, string, string) error {
	return nil
}
func (f *fakeUAPMinIO) FileExists(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f *fakeUAPMinIO) ListFiles(context.Context, *sharedMinio.ListRequest) (*sharedMinio.ListResponse, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) UpdateMetadata(context.Context, string, string, map[string]string) error {
	return nil
}
func (f *fakeUAPMinIO) GetMetadata(context.Context, string, string) (map[string]string, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) UploadAsync(context.Context, *sharedMinio.UploadRequest) (string, error) {
	return "", nil
}
func (f *fakeUAPMinIO) GetUploadStatus(string) (*sharedMinio.UploadProgress, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) WaitForUpload(string, time.Duration) (*sharedMinio.AsyncUploadResult, error) {
	return nil, nil
}
func (f *fakeUAPMinIO) CancelUpload(string) error { return nil }

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read([]byte) (int, error) {
	return 0, e.err
}

func (e errReadCloser) Close() error {
	return nil
}

type noopUAPLogger struct{}

func (noopUAPLogger) Debug(context.Context, ...any)                {}
func (noopUAPLogger) Debugf(context.Context, string, ...any)       {}
func (noopUAPLogger) Info(context.Context, ...any)                 {}
func (noopUAPLogger) Infof(context.Context, string, ...any)        {}
func (noopUAPLogger) Warn(context.Context, ...any)                 {}
func (noopUAPLogger) Warnf(context.Context, string, ...any)        {}
func (noopUAPLogger) Error(context.Context, ...any)                {}
func (noopUAPLogger) Errorf(context.Context, string, ...any)       {}
func (noopUAPLogger) DPanic(context.Context, ...any)               {}
func (noopUAPLogger) DPanicf(context.Context, string, ...any)      {}
func (noopUAPLogger) Panic(context.Context, ...any)                {}
func (noopUAPLogger) Panicf(context.Context, string, ...any)       {}
func (noopUAPLogger) Fatal(context.Context, ...any)                {}
func (noopUAPLogger) Fatalf(context.Context, string, ...any)       {}
func (l noopUAPLogger) WithTrace(context.Context) sharedLog.Logger { return l }

var _ sharedLog.Logger = noopUAPLogger{}

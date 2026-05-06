package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/dryrun"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"

	sharedMinio "github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDryrunSourceTargetHelpers(t *testing.T) {
	errRepo := errors.New("repo")

	tcs := map[string]struct {
		mock func(*datasource.MockUseCase)
		run  func(*implUseCase) error
		err  error
	}{
		"get source error": {
			mock: func(dsUC *datasource.MockUseCase) {
				dsUC.EXPECT().Detail(mock.Anything, testSourceID).Return(datasource.DetailOutput{}, errRepo).Once()
			},
			run: func(uc *implUseCase) error {
				_, err := uc.getSource(context.Background(), testSourceID)
				return err
			},
			err: dryrun.ErrSourceNotFound,
		},
		"get source empty": {
			mock: func(dsUC *datasource.MockUseCase) {
				dsUC.EXPECT().Detail(mock.Anything, testSourceID).Return(datasource.DetailOutput{}, nil).Once()
			},
			run: func(uc *implUseCase) error {
				_, err := uc.getSource(context.Background(), testSourceID)
				return err
			},
			err: dryrun.ErrSourceNotFound,
		},
		"get target error": {
			mock: func(dsUC *datasource.MockUseCase) {
				dsUC.EXPECT().DetailTarget(mock.Anything, datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}).Return(datasource.DetailTargetOutput{}, errRepo).Once()
			},
			run: func(uc *implUseCase) error {
				_, err := uc.getTarget(context.Background(), testSourceID, testTargetID)
				return err
			},
			err: dryrun.ErrTargetNotFound,
		},
		"get target empty": {
			mock: func(dsUC *datasource.MockUseCase) {
				dsUC.EXPECT().DetailTarget(mock.Anything, datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}).Return(datasource.DetailTargetOutput{}, nil).Once()
			},
			run: func(uc *implUseCase) error {
				_, err := uc.getTarget(context.Background(), testSourceID, testTargetID)
				return err
			},
			err: dryrun.ErrTargetNotFound,
		},
		"mark datasource running error": {
			mock: func(dsUC *datasource.MockUseCase) {
				dsUC.EXPECT().MarkDryrunRunning(mock.Anything, datasource.MarkDryrunRunningInput{ID: testSourceID, DryrunLastResultID: "result-1"}).Return(datasource.MarkDryrunRunningOutput{}, errRepo).Once()
			},
			run: func(uc *implUseCase) error {
				_, err := uc.markDatasourceRunning(context.Background(), testSourceID, "result-1")
				return err
			},
			err: dryrun.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, dsUC, _ := newDryrunUC(t)
			tc.mock(dsUC)
			require.ErrorIs(t, tc.run(uc), tc.err)
		})
	}
}

func TestFailDispatch(t *testing.T) {
	errRepo := errors.New("repo")

	tcs := map[string]struct {
		mock   func(*dryrunRepo.MockRepository)
		output model.DryrunResult
		err    error
	}{
		"success": {
			mock: func(r *dryrunRepo.MockRepository) {
				r.EXPECT().CompleteResult(mock.Anything, mock.MatchedBy(func(opt dryrunRepo.CompleteResultOptions) bool {
					return opt.ID == "result-1" && opt.Status == string(model.DryrunStatusFailed) && opt.ErrorMessage == "boom"
				})).Return(dryrunResult(model.DryrunStatusFailed), dryrunSource(model.SourceStatusPending), nil).Once()
			},
			output: dryrunResult(model.DryrunStatusFailed),
		},
		"repo error": {
			mock: func(r *dryrunRepo.MockRepository) {
				r.EXPECT().CompleteResult(mock.Anything, mock.AnythingOfType("repository.CompleteResultOptions")).Return(model.DryrunResult{}, model.DataSource{}, errRepo).Once()
			},
			err: dryrun.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newDryrunUC(t)
			tc.mock(r)

			output, _, err := uc.failDispatch(context.Background(), dryrunResult(model.DryrunStatusRunning), " boom ")

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDownloadArtifactBytes(t *testing.T) {
	errMinIO := errors.New("minio")
	errRead := errors.New("read")

	tcs := map[string]struct {
		input struct {
			bucket string
			path   string
		}
		mock   *fakeDryrunMinIO
		output []byte
		err    bool
	}{
		"nil minio": {
			input: struct {
				bucket string
				path   string
			}{bucket: "bucket", path: "path"},
			err: true,
		},
		"download error": {
			input: struct {
				bucket string
				path   string
			}{bucket: "bucket", path: "path"},
			mock: &fakeDryrunMinIO{downloadErr: errMinIO},
			err:  true,
		},
		"read error": {
			input: struct {
				bucket string
				path   string
			}{bucket: "bucket", path: "path"},
			mock: &fakeDryrunMinIO{reader: errDryrunReadCloser{err: errRead}},
			err:  true,
		},
		"success": {
			input: struct {
				bucket string
				path   string
			}{bucket: "bucket", path: "path"},
			mock:   &fakeDryrunMinIO{body: "artifact"},
			output: []byte("artifact"),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newDryrunUC(t)
			if tc.mock != nil {
				uc.minio = tc.mock
			}

			output, err := uc.downloadArtifactBytes(context.Background(), tc.input.bucket, tc.input.path)

			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDryrunDataHelpers(t *testing.T) {
	uc, _, _, _ := newDryrunUC(t)

	tcs := map[string]struct {
		input  func()
		mock   struct{}
		output struct{}
		err    error
	}{
		"helpers": {
			input: func() {
				require.Nil(t, uc.marshalJSON(func() {}))
				require.Equal(t, 3, uc.parseInt(int32(3)))
				require.Equal(t, 4, uc.parseInt(int64(4)))
				require.Equal(t, 0, uc.parseInt(json.Number("bad")))
				require.Equal(t, intPtr(2), uc.normalizeTotalFound(0, intPtr(2)))
				require.Nil(t, uc.normalizeTotalFound(0, intPtr(0)))
				require.Empty(t, uc.extractKeywords([]string{" ", ""}))
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.input()
		})
	}
}

type fakeDryrunMinIO struct {
	body        string
	reader      io.ReadCloser
	downloadErr error
}

func (f *fakeDryrunMinIO) Connect(context.Context) error { return nil }
func (f *fakeDryrunMinIO) ConnectWithRetry(context.Context, int) error {
	return nil
}
func (f *fakeDryrunMinIO) HealthCheck(context.Context) error { return nil }
func (f *fakeDryrunMinIO) Close() error                      { return nil }
func (f *fakeDryrunMinIO) CreateBucket(context.Context, string) error {
	return nil
}
func (f *fakeDryrunMinIO) DeleteBucket(context.Context, string) error { return nil }
func (f *fakeDryrunMinIO) BucketExists(context.Context, string) (bool, error) {
	return true, nil
}
func (f *fakeDryrunMinIO) ListBuckets(context.Context) ([]*sharedMinio.BucketInfo, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) UploadFile(context.Context, *sharedMinio.UploadRequest) (*sharedMinio.FileInfo, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) GetPresignedUploadURL(context.Context, *sharedMinio.PresignedURLRequest) (*sharedMinio.PresignedURLResponse, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) DownloadFile(context.Context, *sharedMinio.DownloadRequest) (io.ReadCloser, *sharedMinio.DownloadHeaders, error) {
	if f.downloadErr != nil {
		return nil, nil, f.downloadErr
	}
	if f.reader != nil {
		return f.reader, nil, nil
	}
	return io.NopCloser(strings.NewReader(f.body)), nil, nil
}
func (f *fakeDryrunMinIO) StreamFile(ctx context.Context, req *sharedMinio.DownloadRequest) (io.ReadCloser, *sharedMinio.DownloadHeaders, error) {
	return f.DownloadFile(ctx, req)
}
func (f *fakeDryrunMinIO) GetPresignedDownloadURL(context.Context, *sharedMinio.PresignedURLRequest) (*sharedMinio.PresignedURLResponse, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) GetFileInfo(context.Context, string, string) (*sharedMinio.FileInfo, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) DeleteFile(context.Context, string, string) error { return nil }
func (f *fakeDryrunMinIO) CopyFile(context.Context, string, string, string, string) error {
	return nil
}
func (f *fakeDryrunMinIO) MoveFile(context.Context, string, string, string, string) error {
	return nil
}
func (f *fakeDryrunMinIO) FileExists(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f *fakeDryrunMinIO) ListFiles(context.Context, *sharedMinio.ListRequest) (*sharedMinio.ListResponse, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) UpdateMetadata(context.Context, string, string, map[string]string) error {
	return nil
}
func (f *fakeDryrunMinIO) GetMetadata(context.Context, string, string) (map[string]string, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) UploadAsync(context.Context, *sharedMinio.UploadRequest) (string, error) {
	return "", nil
}
func (f *fakeDryrunMinIO) GetUploadStatus(string) (*sharedMinio.UploadProgress, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) WaitForUpload(string, time.Duration) (*sharedMinio.AsyncUploadResult, error) {
	return nil, nil
}
func (f *fakeDryrunMinIO) CancelUpload(string) error { return nil }

type errDryrunReadCloser struct {
	err error
}

func (e errDryrunReadCloser) Read([]byte) (int, error) {
	return 0, e.err
}

func (e errDryrunReadCloser) Close() error {
	return nil
}

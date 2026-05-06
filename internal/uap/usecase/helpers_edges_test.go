package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCommonHelperEdges(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input  func()
		mock   struct{}
		output struct{}
		err    error
	}{
		"primitive helpers": {
			input: func() {
				values := map[string]interface{}{
					"bool_float": float64(1),
					"int":        int(1),
					"int32":      int32(2),
					"int64":      int64(3),
					"float32":    float32(4),
					"float64":    float64(5),
				}
				require.True(t, uc.boolAt(values, "bool_float"))
				require.False(t, uc.boolAt(values, "missing"))
				require.Equal(t, 1, uc.intAt(values, "int"))
				require.Equal(t, 2, uc.intAt(values, "int32"))
				require.Equal(t, 3, uc.intAt(values, "int64"))
				require.Equal(t, 4, uc.intAt(values, "float32"))
				require.Zero(t, uc.intAt(values, "missing"))
				require.Equal(t, float64(4), uc.floatAt(values, "float32"))
				require.Equal(t, float64(1), uc.floatAt(values, "int"))
				require.Zero(t, uc.floatAt(values, "missing"))
				require.Nil(t, uc.stringSliceAt(map[string]interface{}{"items": "bad"}, "items"))
			},
		},
		"collection helpers": {
			input: func() {
				require.Nil(t, uc.chunkRecords(nil))
				require.Nil(t, uc.normalizeStringSlice([]string{" ", ""}))
				require.Nil(t, uc.firstNonEmptySlice([]string{" "}, nil))
				require.Empty(t, uc.extractCrawlKeyword(json.RawMessage(`{`)))
				require.Nil(t, uc.extractLinks("no links"))
				require.Equal(t, []string{"https://example.com/a"}, uc.extractLinks("https://example.com/a, https://example.com/a"))
			},
		},
		"subtitle helpers": {
			input: func() {
				require.Empty(t, uc.resolveSubtitleText(" ", ""))
				require.Empty(t, uc.joinTranscriptSegments(nil))
				require.Empty(t, uc.joinTranscriptSegments([]uap.YouTubeTranscriptSegmentInput{{Text: " "}}))
				require.Nil(t, uc.parseDurationText(""))
				require.Nil(t, uc.parseDurationText("1:2:3:4"))
				require.Nil(t, uc.parseDurationText("1:-2"))
				require.Equal(t, 3723, *uc.parseDurationText("1:02:03"))
			},
		},
		"metadata helpers": {
			input: func() {
				raw := uc.mergeRawMetadata(json.RawMessage(`{`), nil, 0, nil)
				require.Contains(t, string(raw), uap.ArtifactsMetadataKey)
				require.Equal(t, 1.5, *uc.float64Ptr(1.5))
			},
		},
		"marshal errors": {
			input: func() {
				badRecord := uap.UAPRecord{
					Identity:     uap.UAPIdentity{UAPID: "bad"},
					PlatformMeta: map[string]interface{}{"bad": func() {}},
				}
				_, err := uc.marshalChunkJSONL([]uap.UAPRecord{badRecord})
				require.Error(t, err)
				_, err = uc.uploadChunk(context.Background(), &fakeUAPMinIO{}, "bucket", "project", "source", "batch", 1, []uap.UAPRecord{badRecord})
				require.Error(t, err)
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.input()
		})
	}
}

func TestDownloadSubtitleTextEdges(t *testing.T) {
	tcs := map[string]struct {
		input  string
		server func(http.ResponseWriter, *http.Request)
		output string
		err    bool
	}{
		"empty url": {err: true},
		"plain text": {
			server: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(" hello   world "))
			},
			output: "hello world",
		},
		"bad request url": {input: "http://[::1", err: true},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := &implUseCase{}
			input := tc.input
			if tc.server != nil {
				server := httptest.NewServer(http.HandlerFunc(tc.server))
				defer server.Close()
				input = server.URL
			}

			output, err := uc.downloadSubtitleText(input)

			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestSubtitleHTTPClientEdges(t *testing.T) {
	tcs := map[string]struct {
		client *http.Client
		output string
		err    bool
	}{
		"client error": {
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("dial")
			})},
			err: true,
		},
		"body read error": {
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{err: errors.New("read")}}, nil
			})},
			err: true,
		},
		"resolve logs and skips duplicate": {
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("dial")
			})},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := &implUseCase{subtitleHTTPClient: tc.client, l: noopUAPLogger{}}
			if name == "resolve logs and skips duplicate" {
				require.Empty(t, uc.resolveSubtitleText("http://example.com/a", "http://example.com/a"))
				return
			}

			output, err := uc.downloadSubtitleText("http://example.com/subtitle")

			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestParseAndStoreRawBatchEdges(t *testing.T) {
	baseInput := uap.ParseAndStoreRawBatchInput{
		RawBatchID:    "raw-batch-1",
		ProjectID:     "project-1",
		SourceID:      "source-1",
		TaskID:        "task-1",
		Platform:      "youtube",
		Action:        "full_flow",
		StorageBucket: "raw-bucket",
		StoragePath:   "raw/path.json",
		BatchID:       "batch-1",
	}

	tcs := map[string]struct {
		setup func(*implUseCase, *repo.MockRepository)
		err   error
	}{
		"nil parser": {
			setup: func(uc *implUseCase, _ *repo.MockRepository) {
				uc.parsers[normalizeParseKey("youtube", "full_flow")] = nil
			},
			err: uap.ErrInvalidRawBatchInput,
		},
		"empty output bucket falls back": {
			setup: func(uc *implUseCase, r *repo.MockRepository) {
				uc.outputBucket = ""
				uc.minio = &fakeUAPMinIO{downloadBody: `{"result":{"videos":[{"video":{"video_id":"yt-1"}}]}}`}
				r.EXPECT().ClaimRawBatchForParsing(mock.Anything, "raw-batch-1").Return(true, nil).Once()
				r.EXPECT().MarkRawBatchDownloaded(mock.Anything, repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"}).Return(nil).Once()
				r.EXPECT().MarkRawBatchParsed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchParsedOptions) bool {
					return opt.RawBatchID == "raw-batch-1" && opt.PublishRecordCount == 1
				})).Return(nil).Once()
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r := repo.NewMockRepository(t)
			uc := New(noopUAPLogger{}, r, &fakeUAPMinIO{}, "parsed-bucket", nil).(*implUseCase)
			tc.setup(uc, r)

			err := uc.ParseAndStoreRawBatch(context.Background(), baseInput)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

var _ io.Reader = errReadCloser{}

func TestFailRawBatch(t *testing.T) {
	errRepo := errors.New("repo")

	tcs := map[string]struct {
		mock func(*repo.MockRepository)
		err  error
	}{
		"success": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.MatchedBy(func(opt repo.MarkRawBatchFailedOptions) bool {
					return opt.RawBatchID == "raw-batch-1" && opt.ErrorMessage == "boom" && opt.PublishError == "publish"
				})).Return(nil).Once()
			},
		},
		"repo error": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().MarkRawBatchFailed(mock.Anything, mock.AnythingOfType("repository.MarkRawBatchFailedOptions")).Return(errRepo).Once()
			},
			err: errRepo,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r := repo.NewMockRepository(t)
			tc.mock(r)
			uc := &implUseCase{repo: r, l: noopUAPLogger{}}

			err := uc.failRawBatch(context.Background(), uap.ParseAndStoreRawBatchInput{RawBatchID: "raw-batch-1"}, "boom", "publish", nil, 0, nil)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

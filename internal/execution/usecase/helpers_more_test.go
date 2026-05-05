package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/microservice"

	sharedMinio "github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildDispatchSpecsMore(t *testing.T) {
	uc, _, _, _ := newExecutionUC(t)

	tcs := map[string]struct {
		input struct {
			source model.DataSource
			target model.CrawlTarget
		}
		mock   struct{}
		output int
		err    error
	}{
		"tiktok keyword": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: executionSource(), target: executionTarget()},
			output: 1,
		},
		"facebook keyword": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: func() model.DataSource {
				source := executionSource()
				source.SourceType = model.SourceTypeFacebook
				return source
			}(), target: executionTarget()},
			output: 1,
		},
		"youtube keyword": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: func() model.DataSource {
				source := executionSource()
				source.SourceType = model.SourceTypeYouTube
				return source
			}(), target: executionTarget()},
			output: 1,
		},
		"empty keyword": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: executionSource(), target: func() model.CrawlTarget {
				target := executionTarget()
				target.Values = []string{" "}
				return target
			}()},
			err: execution.ErrDispatchNotAllowed,
		},
		"facebook post url": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: func() model.DataSource {
				source := executionSource()
				source.SourceType = model.SourceTypeFacebook
				return source
			}(), target: func() model.CrawlTarget {
				target := executionTarget()
				target.TargetType = model.TargetTypePostURL
				target.PlatformMeta = json.RawMessage(`{"parse_ids":["123"," 456 "]}`)
				return target
			}()},
			output: 1,
		},
		"facebook post url invalid meta": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: func() model.DataSource {
				source := executionSource()
				source.SourceType = model.SourceTypeFacebook
				return source
			}(), target: func() model.CrawlTarget {
				target := executionTarget()
				target.TargetType = model.TargetTypePostURL
				return target
			}()},
			err: execution.ErrPlatformMetaParseIDs,
		},
		"unsupported": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: func() model.DataSource {
				source := executionSource()
				source.SourceType = model.SourceTypeWebhook
				return source
			}(), target: executionTarget()},
			err: execution.ErrUnsupportedDispatchMapping,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output, err := uc.buildDispatchSpecs(tc.input.source, tc.input.target)
			require.ErrorIs(t, err, tc.err)
			require.Len(t, output, tc.output)
		})
	}
}

func TestExecutionHelperEdges(t *testing.T) {
	uc, _, _, _ := newExecutionUC(t)

	t.Run("validate dispatch context", func(t *testing.T) {
		ctx := repo.DispatchContext{Source: executionSource(), Target: executionTarget()}
		ctx.Source.SourceCategory = model.SourceCategoryPassive
		require.ErrorIs(t, uc.validateDispatchContext(ctx), execution.ErrDispatchNotAllowed)

		ctx = repo.DispatchContext{Source: executionSource(), Target: executionTarget()}
		ctx.Source.Status = model.SourceStatusArchived
		require.ErrorIs(t, uc.validateDispatchContext(ctx), execution.ErrDispatchNotAllowed)

		ctx = repo.DispatchContext{Source: executionSource(), Target: executionTarget()}
		ctx.Source.Status = model.SourceStatusPaused
		require.ErrorIs(t, uc.validateScheduledDispatchContext(ctx), execution.ErrDispatchNotAllowed)
	})

	t.Run("parse int64", func(t *testing.T) {
		require.Equal(t, int64Ptr(1), uc.parseInt64(1))
		require.Equal(t, int64Ptr(2), uc.parseInt64(int32(2)))
		require.Equal(t, int64Ptr(3), uc.parseInt64(int64(3)))
		require.Equal(t, int64Ptr(4), uc.parseInt64(float64(4.9)))
		require.Nil(t, uc.parseInt64(json.Number("bad")))
		require.Nil(t, uc.parseInt64("bad"))
	})

	t.Run("metadata marshal", func(t *testing.T) {
		got, err := uc.marshalMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, got)

		_, err = uc.marshalMetadata(map[string]interface{}{"bad": func() {}})
		require.Error(t, err)
	})

	t.Run("completion validation", func(t *testing.T) {
		require.NoError(t, uc.validateCompletionInput(execution.HandleCompletionInput{TaskID: "task-1", Status: "error"}))
		require.ErrorIs(t, uc.validateCompletionInput(execution.HandleCompletionInput{TaskID: "task-1", Status: "success"}), execution.ErrInvalidCompletionInput)
		require.ErrorIs(t, uc.validateCompletionInput(execution.HandleCompletionInput{TaskID: "task-1", Status: "unknown"}), execution.ErrInvalidCompletionInput)
	})

	t.Run("facebook parse ids", func(t *testing.T) {
		_, err := uc.parseFacebookParseIDs(json.RawMessage(`{"parse_ids":[1]}`))
		require.ErrorIs(t, err, execution.ErrPlatformMetaParseIDs)
		_, err = uc.parseFacebookParseIDs(json.RawMessage(`{"parse_ids":[" "]}`))
		require.ErrorIs(t, err, execution.ErrPlatformMetaParseIDs)
	})
}

func TestPublishDispatch(t *testing.T) {
	tcs := map[string]struct {
		nil bool
		err bool
	}{
		"nil publisher": {nil: true, err: true},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newExecutionUC(t)
			if tc.nil {
				uc.publisher = nil
			}

			err := uc.publishDispatch(context.Background(), execution.PublishDispatchInput{})
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestVerifyMinIOObject(t *testing.T) {
	errMinIO := errors.New("minio")

	tcs := map[string]struct {
		minio  interface{}
		output *sharedMinio.FileInfo
		err    bool
	}{
		"nil minio": {err: true},
		"exists stat success": {
			minio:  fakeMinIO{exists: true, info: &sharedMinio.FileInfo{Size: 10}},
			output: &sharedMinio.FileInfo{Size: 10},
		},
		"exists stat error": {
			minio: fakeMinIO{exists: true, err: errMinIO},
			err:   true,
		},
		"missing": {
			minio: fakeMinIO{exists: false},
			err:   true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newExecutionUC(t)
			if client, ok := tc.minio.(fakeMinIO); ok {
				uc.minio = client
			}

			output, err := uc.verifyMinIOObject(context.Background(), "bucket", "path")
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestResolveProjectDomainTypeCodeEdges(t *testing.T) {
	tcs := map[string]struct {
		mock func(*microservice.MockProjectUseCase)
		nil  bool
		err  bool
	}{
		"nil project": {nil: true, err: true},
		"project error": {
			mock: func(project *microservice.MockProjectUseCase) {
				project.EXPECT().Detail(mock.Anything, testProjectID).Return(microservice.ProjectDetail{}, errors.New("project")).Once()
			},
			err: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, project := newExecutionUC(t)
			if tc.nil {
				uc.project = nil
			} else if tc.mock != nil {
				tc.mock(project)
			}

			_, err := uc.resolveProjectDomainTypeCode(context.Background(), testProjectID)
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewSetsDefaults(t *testing.T) {
	uc, r, pub, project := newExecutionUC(t)
	got := New(uc.l, r, nil, pub, nil, project)
	impl, ok := got.(*implUseCase)
	require.True(t, ok)
	require.NotNil(t, impl.now)
	require.NotNil(t, impl.sleep)
}

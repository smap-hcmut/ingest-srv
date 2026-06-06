package usecase

import (
	"encoding/json"
	"testing"

	"ingest-srv/internal/dryrun"
	"ingest-srv/internal/model"
)

func TestBuildDispatchSpecKeywordFullFlowPlatforms(t *testing.T) {
	uc := &implUseCase{}
	target := &model.CrawlTarget{
		TargetType: model.TargetTypeKeyword,
		Values:     []string{"bia heineken", "bia tiger"},
	}

	tests := []struct {
		name       string
		sourceType model.SourceType
		queue      dryrun.QueueName
	}{
		{
			name:       "tiktok keyword",
			sourceType: model.SourceTypeTikTok,
			queue:      dryrun.QueueNameTikTokTasks,
		},
		{
			name:       "facebook keyword",
			sourceType: model.SourceTypeFacebook,
			queue:      dryrun.QueueNameFacebookTasks,
		},
		{
			name:       "youtube keyword",
			sourceType: model.SourceTypeYouTube,
			queue:      dryrun.QueueNameYouTubeTasks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, warnings, err := uc.buildDispatchSpec(model.DataSource{
				SourceType:     tt.sourceType,
				SourceCategory: model.SourceCategoryCrawl,
			}, target, 5)
			if err != nil {
				t.Fatalf("buildDispatchSpec returned error: %v", err)
			}
			if spec.Queue != tt.queue {
				t.Fatalf("unexpected queue: got %s want %s", spec.Queue, tt.queue)
			}
			if spec.Action != dryrun.ActionNameFullFlow {
				t.Fatalf("unexpected action: got %s want %s", spec.Action, dryrun.ActionNameFullFlow)
			}
			if spec.Params["keyword"] != "bia heineken" {
				t.Fatalf("unexpected keyword param: %#v", spec.Params["keyword"])
			}
			if spec.Params["limit"] != 5 {
				t.Fatalf("unexpected limit param: %#v", spec.Params["limit"])
			}
			if spec.Params[dryrun.ParamKeyRuntimeKind] != string(dryrun.RuntimeKindDryrun) {
				t.Fatalf("unexpected runtime kind: %#v", spec.Params[dryrun.ParamKeyRuntimeKind])
			}
			if spec.Params[dryrun.ParamKeyDryrunWarningCode] != string(dryrun.WarningCodeMultiValueKeyword) {
				t.Fatalf("unexpected warning code param: %#v", spec.Params[dryrun.ParamKeyDryrunWarningCode])
			}

			var parsedWarnings []dryrun.Warning
			if err := json.Unmarshal(warnings, &parsedWarnings); err != nil {
				t.Fatalf("warning payload is not valid JSON: %v", err)
			}
			if len(parsedWarnings) != 1 {
				t.Fatalf("unexpected warning count: %d", len(parsedWarnings))
			}
			if parsedWarnings[0].Code != dryrun.WarningCodeMultiValueKeyword {
				t.Fatalf("unexpected warning code: %s", parsedWarnings[0].Code)
			}
		})
	}
}

func TestBuildDispatchSpecFacebookPostURL(t *testing.T) {
	uc := &implUseCase{}
	target := &model.CrawlTarget{
		TargetType:   model.TargetTypePostURL,
		PlatformMeta: json.RawMessage(`{"parse_ids":["123","456"]}`),
	}

	spec, warnings, err := uc.buildDispatchSpec(model.DataSource{
		SourceType:     model.SourceTypeFacebook,
		SourceCategory: model.SourceCategoryCrawl,
	}, target, 3)
	if err != nil {
		t.Fatalf("buildDispatchSpec returned error: %v", err)
	}
	if spec.Queue != dryrun.QueueNameFacebookTasks {
		t.Fatalf("unexpected queue: %s", spec.Queue)
	}
	if spec.Action != dryrun.ActionNamePostDetail {
		t.Fatalf("unexpected action: %s", spec.Action)
	}
	if warnings != nil {
		t.Fatalf("unexpected warnings: %s", string(warnings))
	}
	parseIDs, ok := spec.Params["parse_ids"].([]string)
	if !ok {
		t.Fatalf("parse_ids param has wrong type: %#v", spec.Params["parse_ids"])
	}
	if len(parseIDs) != 2 || parseIDs[0] != "123" || parseIDs[1] != "456" {
		t.Fatalf("unexpected parse_ids: %#v", parseIDs)
	}
}

func TestBuildDispatchSpecUnsupportedProfileTarget(t *testing.T) {
	uc := &implUseCase{}
	target := &model.CrawlTarget{
		TargetType: model.TargetTypeProfile,
		Values:     []string{"https://facebook.com/example"},
	}

	_, _, err := uc.buildDispatchSpec(model.DataSource{
		SourceType:     model.SourceTypeFacebook,
		SourceCategory: model.SourceCategoryCrawl,
	}, target, 3)
	if err != dryrun.ErrUnsupportedMapping {
		t.Fatalf("unexpected error: got %v want %v", err, dryrun.ErrUnsupportedMapping)
	}
}

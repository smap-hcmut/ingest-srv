package usecase

import (
	"encoding/json"

	"ingest-srv/internal/model"
)

type executionInput struct {
	source model.DataSource
	target *model.CrawlTarget
}

type executionResult struct {
	status       model.DryrunStatus
	warnings     json.RawMessage
	errorMessage string
}

type localExecutor struct{}

func (localExecutor) Execute(input executionInput) executionResult {
	if input.source.SourceCategory == model.SourceCategoryCrawl {
		if input.source.CrawlMode == nil || input.source.CrawlIntervalMinutes == nil || *input.source.CrawlIntervalMinutes <= 0 {
			return executionResult{
				status:       model.DryrunStatusFailed,
				errorMessage: "crawl source missing required crawl configuration",
			}
		}
		if input.target == nil || !input.target.IsActive {
			return executionResult{
				status:       model.DryrunStatusFailed,
				errorMessage: "crawl target must exist and be active",
			}
		}
		if len(input.target.Values) == 0 {
			return executionResult{
				status:       model.DryrunStatusFailed,
				errorMessage: "crawl target group must contain at least one value",
			}
		}
	}
	if input.source.SourceCategory == model.SourceCategoryPassive {
		if input.source.SourceType == model.SourceTypeWebhook {
			if input.source.WebhookID == "" || input.source.WebhookSecretEncrypted == "" {
				return executionResult{
					status:       model.DryrunStatusFailed,
					errorMessage: "webhook source missing required webhook configuration",
				}
			}
		}
	}

	warnings, _ := json.Marshal([]map[string]string{
		{
			"code":    "control_plane_only_no_remote_execution",
			"message": "Dryrun validation passed without remote execution",
		},
	})

	return executionResult{
		status:   model.DryrunStatusWarning,
		warnings: warnings,
	}
}

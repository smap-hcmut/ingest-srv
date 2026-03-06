package usecase

import (
	"net/url"
	"ingest-srv/internal/datasource"
	"ingest-srv/internal/model"
	"strings"
)

// validateSourceType checks if the given source_type is valid.
func (uc *implUseCase) validateSourceType(st string) error {
	switch model.SourceType(st) {
	case model.SourceTypeTikTok, model.SourceTypeFacebook, model.SourceTypeYouTube,
		model.SourceTypeFileUpload, model.SourceTypeWebhook:
		return nil
	default:
		return datasource.ErrInvalidSourceType
	}
}

// validateSourceCategory checks if the given source_category is valid.
func (uc *implUseCase) validateSourceCategory(cat string) error {
	switch model.SourceCategory(cat) {
	case model.SourceCategoryCrawl, model.SourceCategoryPassive:
		return nil
	default:
		return datasource.ErrInvalidCategory
	}
}

// validateCrawlMode checks if the given crawl_mode is valid.
func (uc *implUseCase) validateCrawlMode(mode string) error {
	switch model.CrawlMode(mode) {
	case model.CrawlModeSleep, model.CrawlModeNormal, model.CrawlModeCrisis:
		return nil
	default:
		return datasource.ErrInvalidCrawlMode
	}
}

// validateTriggerType checks if the given trigger_type is valid.
func (uc *implUseCase) validateTriggerType(triggerType string) error {
	switch model.TriggerType(triggerType) {
	case model.TriggerTypeManual, model.TriggerTypeScheduled, model.TriggerTypeProjectEvent,
		model.TriggerTypeCrisisEvent, model.TriggerTypeWebhookPush:
		return nil
	default:
		return datasource.ErrCrawlModeNotAllowed
	}
}

// inferCategory derives source_category from source_type.
// CRAWL: TIKTOK, FACEBOOK, YOUTUBE
// PASSIVE: FILE_UPLOAD, WEBHOOK
func (uc *implUseCase) inferCategory(sourceType string) string {
	switch model.SourceType(sourceType) {
	case model.SourceTypeFileUpload, model.SourceTypeWebhook:
		return string(model.SourceCategoryPassive)
	default:
		return string(model.SourceCategoryCrawl)
	}
}

// validateTargetType checks if the given target_type is valid.
func (uc *implUseCase) validateTargetType(tt string) error {
	switch model.TargetType(tt) {
	case model.TargetTypeKeyword, model.TargetTypeProfile, model.TargetTypePostURL:
		return nil
	default:
		return datasource.ErrInvalidTargetType
	}
}

func normalizeTargetValues(values []string, dedupe bool) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if dedupe {
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
		}
		out = append(out, trimmed)
	}
	return out
}

func validateTargetURLValues(values []string) error {
	for _, value := range values {
		parsed, err := url.ParseRequestURI(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return datasource.ErrTargetValuesMustBeURLs
		}
	}
	return nil
}

func (uc *implUseCase) prepareTargetValues(targetType model.TargetType, values []string) ([]string, error) {
	normalized := normalizeTargetValues(values, targetType == model.TargetTypeKeyword)
	if len(normalized) == 0 {
		return nil, datasource.ErrTargetValuesRequired
	}
	switch targetType {
	case model.TargetTypeProfile, model.TargetTypePostURL:
		if err := validateTargetURLValues(normalized); err != nil {
			return nil, err
		}
	}
	return normalized, nil
}

func (uc *implUseCase) validCreateTargetGroupInput(input datasource.CreateTargetGroupInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" {
		return datasource.ErrProjectIDRequired
	}
	if input.CrawlIntervalMinutes <= 0 {
		return datasource.ErrInvalidTargetInterval
	}
	return nil
}

func (uc *implUseCase) validDetailTargetInput(input datasource.DetailTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	return nil
}

func (uc *implUseCase) validListTargetsInput(input datasource.ListTargetsInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" {
		return datasource.ErrProjectIDRequired
	}
	if strings.TrimSpace(input.TargetType) != "" {
		if err := uc.validateTargetType(strings.TrimSpace(input.TargetType)); err != nil {
			return err
		}
	}
	return nil
}

func (uc *implUseCase) validUpdateTargetInput(input datasource.UpdateTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	if input.CrawlIntervalMinutes != nil && *input.CrawlIntervalMinutes <= 0 {
		return datasource.ErrInvalidTargetInterval
	}
	return nil
}

func (uc *implUseCase) validDeleteTargetInput(input datasource.DeleteTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	return nil
}

func (uc *implUseCase) validActivateInput(input datasource.ActivateInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return datasource.ErrNotFound
	}
	return nil
}

func (uc *implUseCase) validPauseInput(input datasource.PauseInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return datasource.ErrNotFound
	}
	return nil
}

func (uc *implUseCase) validResumeInput(input datasource.ResumeInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return datasource.ErrNotFound
	}
	return nil
}

func (uc *implUseCase) validUpdateCrawlModeInput(input datasource.UpdateCrawlModeInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return datasource.ErrNotFound
	}
	if err := uc.validateCrawlMode(strings.TrimSpace(input.CrawlMode)); err != nil {
		return err
	}
	if err := uc.validateTriggerType(strings.TrimSpace(input.TriggerType)); err != nil {
		return err
	}
	return nil
}

package usecase

import (
	"ingest-srv/internal/datasource"
	"ingest-srv/internal/model"
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

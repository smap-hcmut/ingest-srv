package datasource

import "errors"

// Domain errors — returned by UseCase layer.
var (
	ErrNotFound            = errors.New("data source not found")
	ErrNameRequired        = errors.New("data source name is required")
	ErrProjectIDRequired   = errors.New("project_id is required")
	ErrSourceTypeRequired  = errors.New("source_type is required")
	ErrInvalidSourceType   = errors.New("invalid source_type")
	ErrInvalidCategory     = errors.New("invalid source_category")
	ErrInvalidCrawlMode    = errors.New("invalid crawl_mode")
	ErrCrawlConfigRequired = errors.New("crawl source requires crawl_mode and crawl_interval_minutes > 0")
	ErrCreateFailed        = errors.New("failed to create data source")
	ErrUpdateFailed        = errors.New("failed to update data source")
	ErrDeleteFailed        = errors.New("failed to delete data source")
	ErrListFailed          = errors.New("failed to list data sources")
	ErrUpdateNotAllowed    = errors.New("cannot update config/mapping on an active source")

	// CrawlTarget errors.
	ErrTargetNotFound      = errors.New("crawl target not found")
	ErrTargetValueRequired = errors.New("crawl target value is required")
	ErrInvalidTargetType   = errors.New("invalid target_type; must be KEYWORD, PROFILE, or POST_URL")
	ErrSourceNotCrawl      = errors.New("crawl targets can only be added to CRAWL sources")
	ErrTargetCreateFailed  = errors.New("failed to create crawl target")
	ErrTargetUpdateFailed  = errors.New("failed to update crawl target")
	ErrTargetDeleteFailed  = errors.New("failed to delete crawl target")
	ErrTargetListFailed    = errors.New("failed to list crawl targets")
)

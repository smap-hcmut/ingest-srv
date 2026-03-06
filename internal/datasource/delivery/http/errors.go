package http

import (
	"ingest-srv/internal/datasource"
	pkgErrors "ingest-srv/pkg/errors"
)

// Delivery-layer HTTP errors — domain-specific error codes in 170xxx range.
var (
	errNotFound            = pkgErrors.NewHTTPError(170001, "Data source not found")
	errNameRequired        = pkgErrors.NewHTTPError(170002, "Data source name is required")
	errProjectIDRequired   = pkgErrors.NewHTTPError(170003, "Project ID is required")
	errSourceTypeRequired  = pkgErrors.NewHTTPError(170004, "Source type is required")
	errInvalidSourceType   = pkgErrors.NewHTTPError(170005, "Invalid source type")
	errInvalidCategory     = pkgErrors.NewHTTPError(170006, "Invalid source category")
	errInvalidCrawlMode    = pkgErrors.NewHTTPError(170007, "Invalid crawl mode")
	errCrawlConfigRequired = pkgErrors.NewHTTPError(170008, "Crawl source requires crawl_mode and crawl_interval_minutes")
	errCreateFailed        = pkgErrors.NewHTTPError(170009, "Failed to create data source")
	errUpdateFailed        = pkgErrors.NewHTTPError(170010, "Failed to update data source")
	errDeleteFailed        = pkgErrors.NewHTTPError(170011, "Failed to delete data source")
	errListFailed          = pkgErrors.NewHTTPError(170012, "Failed to list data sources")
	errUpdateNotAllowed    = pkgErrors.NewHTTPError(170013, "Cannot update config/mapping on an active source")
	errWrongBody           = pkgErrors.NewHTTPError(170014, "Wrong request body")
	errInvalidCrawlInterval = pkgErrors.NewHTTPError(170015, "Invalid crawl_interval_minutes; must be greater than 0")
	errInvalidTransition   = pkgErrors.NewHTTPError(170016, "Invalid datasource lifecycle transition")
	errActivateNotAllowed  = pkgErrors.NewHTTPError(170017, "Datasource cannot be activated in its current state")
	errPauseNotAllowed     = pkgErrors.NewHTTPError(170018, "Datasource cannot be paused in its current state")
	errResumeNotAllowed    = pkgErrors.NewHTTPError(170019, "Datasource cannot be resumed in its current state")
	errCrawlModeNotAllowed = pkgErrors.NewHTTPError(170020, "Crawl mode update is not allowed for this datasource")
	errInternal            = &pkgErrors.HTTPError{Code: 170999, Message: "Internal server error", StatusCode: 500}

	// CrawlTarget errors — 170100+ range.
	errTargetNotFound        = pkgErrors.NewHTTPError(170101, "Crawl target not found")
	errTargetValuesRequired = pkgErrors.NewHTTPError(170102, "Crawl target values are required")
	errInvalidTargetType    = pkgErrors.NewHTTPError(170103, "Invalid target_type; must be KEYWORD, PROFILE, or POST_URL")
	errSourceNotCrawl       = pkgErrors.NewHTTPError(170104, "Crawl targets can only be added to CRAWL sources")
	errTargetCreateFailed   = pkgErrors.NewHTTPError(170105, "Failed to create crawl target")
	errTargetUpdateFailed   = pkgErrors.NewHTTPError(170106, "Failed to update crawl target")
	errTargetDeleteFailed   = pkgErrors.NewHTTPError(170107, "Failed to delete crawl target")
	errTargetListFailed     = pkgErrors.NewHTTPError(170108, "Failed to list crawl targets")
	errInvalidTargetInterval = pkgErrors.NewHTTPError(170109, "Invalid crawl_interval_minutes; must be greater than 0")
	errTargetValuesMustBeURLs = pkgErrors.NewHTTPError(170110, "Crawl target values must be valid URLs")
)

// mapError maps UseCase domain errors to delivery HTTP errors.
func (h *handler) mapError(err error) error {
	switch err {
	case datasource.ErrNotFound:
		return errNotFound
	case datasource.ErrNameRequired:
		return errNameRequired
	case datasource.ErrProjectIDRequired:
		return errProjectIDRequired
	case datasource.ErrSourceTypeRequired:
		return errSourceTypeRequired
	case datasource.ErrInvalidSourceType:
		return errInvalidSourceType
	case datasource.ErrInvalidCategory:
		return errInvalidCategory
	case datasource.ErrInvalidCrawlMode:
		return errInvalidCrawlMode
	case datasource.ErrCrawlConfigRequired:
		return errCrawlConfigRequired
	case datasource.ErrCreateFailed:
		return errCreateFailed
	case datasource.ErrUpdateFailed:
		return errUpdateFailed
	case datasource.ErrDeleteFailed:
		return errDeleteFailed
	case datasource.ErrListFailed:
		return errListFailed
	case datasource.ErrUpdateNotAllowed:
		return errUpdateNotAllowed
	case datasource.ErrInvalidTransition:
		return errInvalidTransition
	case datasource.ErrActivateNotAllowed:
		return errActivateNotAllowed
	case datasource.ErrPauseNotAllowed:
		return errPauseNotAllowed
	case datasource.ErrResumeNotAllowed:
		return errResumeNotAllowed
	case datasource.ErrCrawlModeNotAllowed:
		return errCrawlModeNotAllowed

	// CrawlTarget errors
	case datasource.ErrTargetNotFound:
		return errTargetNotFound
	case datasource.ErrTargetValuesRequired:
		return errTargetValuesRequired
	case datasource.ErrTargetValuesMustBeURLs:
		return errTargetValuesMustBeURLs
	case datasource.ErrInvalidTargetType:
		return errInvalidTargetType
	case datasource.ErrSourceNotCrawl:
		return errSourceNotCrawl
	case datasource.ErrTargetCreateFailed:
		return errTargetCreateFailed
	case datasource.ErrTargetUpdateFailed:
		return errTargetUpdateFailed
	case datasource.ErrTargetDeleteFailed:
		return errTargetDeleteFailed
	case datasource.ErrTargetListFailed:
		return errTargetListFailed
	case datasource.ErrInvalidTargetInterval:
		return errInvalidTargetInterval
	default:
		return errInternal
	}
}

package http

import (
	"errors"
	"net/http"

	"ingest-srv/internal/datasource"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors — sequential codes per domain.
var (
	errNotFound             = &pkgErrors.HTTPError{Code: 1, Message: "Data source not found", StatusCode: http.StatusNotFound}
	errNameRequired         = &pkgErrors.HTTPError{Code: 2, Message: "Data source name is required", StatusCode: http.StatusBadRequest}
	errProjectIDRequired    = &pkgErrors.HTTPError{Code: 3, Message: "Project ID is required", StatusCode: http.StatusBadRequest}
	errSourceTypeRequired   = &pkgErrors.HTTPError{Code: 4, Message: "Source type is required", StatusCode: http.StatusBadRequest}
	errInvalidSourceType    = &pkgErrors.HTTPError{Code: 5, Message: "Invalid source type", StatusCode: http.StatusBadRequest}
	errInvalidCategory      = &pkgErrors.HTTPError{Code: 6, Message: "Invalid source category", StatusCode: http.StatusBadRequest}
	errInvalidCrawlMode     = &pkgErrors.HTTPError{Code: 7, Message: "Invalid crawl mode", StatusCode: http.StatusBadRequest}
	errCrawlConfigRequired  = &pkgErrors.HTTPError{Code: 8, Message: "Crawl source requires crawl_mode and crawl_interval_minutes", StatusCode: http.StatusBadRequest}
	errCreateFailed         = &pkgErrors.HTTPError{Code: 9, Message: "Failed to create data source", StatusCode: http.StatusInternalServerError}
	errUpdateFailed         = &pkgErrors.HTTPError{Code: 10, Message: "Failed to update data source", StatusCode: http.StatusInternalServerError}
	errDeleteFailed         = &pkgErrors.HTTPError{Code: 11, Message: "Failed to delete data source", StatusCode: http.StatusInternalServerError}
	errListFailed           = &pkgErrors.HTTPError{Code: 12, Message: "Failed to list data sources", StatusCode: http.StatusInternalServerError}
	errUpdateNotAllowed     = &pkgErrors.HTTPError{Code: 13, Message: "Cannot update config/mapping on an active source", StatusCode: http.StatusConflict}
	errWrongBody            = &pkgErrors.HTTPError{Code: 14, Message: "Wrong request body", StatusCode: http.StatusBadRequest}
	errInvalidCrawlInterval = &pkgErrors.HTTPError{Code: 15, Message: "Invalid crawl_interval_minutes; must be greater than 0", StatusCode: http.StatusBadRequest}
	errInvalidTransition    = &pkgErrors.HTTPError{Code: 16, Message: "Invalid datasource lifecycle transition", StatusCode: http.StatusBadRequest}
	errActivateNotAllowed   = &pkgErrors.HTTPError{Code: 17, Message: "Datasource cannot be activated in its current state", StatusCode: http.StatusConflict}
	errPauseNotAllowed      = &pkgErrors.HTTPError{Code: 18, Message: "Datasource cannot be paused in its current state", StatusCode: http.StatusConflict}
	errResumeNotAllowed     = &pkgErrors.HTTPError{Code: 19, Message: "Datasource cannot be resumed in its current state", StatusCode: http.StatusConflict}
	errCrawlModeNotAllowed  = &pkgErrors.HTTPError{Code: 20, Message: "Crawl mode update is not allowed for this datasource", StatusCode: http.StatusConflict}
	errInternal             = &pkgErrors.HTTPError{Code: 99, Message: "Internal server error", StatusCode: http.StatusInternalServerError}

	// CrawlTarget errors — 101+ range.
	errTargetNotFound         = &pkgErrors.HTTPError{Code: 101, Message: "Crawl target not found", StatusCode: http.StatusNotFound}
	errTargetValuesRequired   = &pkgErrors.HTTPError{Code: 102, Message: "Crawl target values are required", StatusCode: http.StatusBadRequest}
	errInvalidTargetType      = &pkgErrors.HTTPError{Code: 103, Message: "Invalid target_type; must be KEYWORD, PROFILE, or POST_URL", StatusCode: http.StatusBadRequest}
	errSourceNotCrawl         = &pkgErrors.HTTPError{Code: 104, Message: "Crawl targets can only be added to CRAWL sources", StatusCode: http.StatusBadRequest}
	errTargetCreateFailed     = &pkgErrors.HTTPError{Code: 105, Message: "Failed to create crawl target", StatusCode: http.StatusInternalServerError}
	errTargetUpdateFailed     = &pkgErrors.HTTPError{Code: 106, Message: "Failed to update crawl target", StatusCode: http.StatusInternalServerError}
	errTargetDeleteFailed     = &pkgErrors.HTTPError{Code: 107, Message: "Failed to delete crawl target", StatusCode: http.StatusInternalServerError}
	errTargetListFailed       = &pkgErrors.HTTPError{Code: 108, Message: "Failed to list crawl targets", StatusCode: http.StatusInternalServerError}
	errInvalidTargetInterval  = &pkgErrors.HTTPError{Code: 109, Message: "Invalid crawl_interval_minutes; must be greater than 0", StatusCode: http.StatusBadRequest}
	errTargetValuesMustBeURLs = &pkgErrors.HTTPError{Code: 110, Message: "Crawl target values must be valid URLs", StatusCode: http.StatusBadRequest}
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, datasource.ErrNotFound):
		return errNotFound
	case errors.Is(err, datasource.ErrNameRequired):
		return errNameRequired
	case errors.Is(err, datasource.ErrProjectIDRequired):
		return errProjectIDRequired
	case errors.Is(err, datasource.ErrSourceTypeRequired):
		return errSourceTypeRequired
	case errors.Is(err, datasource.ErrInvalidSourceType):
		return errInvalidSourceType
	case errors.Is(err, datasource.ErrInvalidCategory):
		return errInvalidCategory
	case errors.Is(err, datasource.ErrInvalidCrawlMode):
		return errInvalidCrawlMode
	case errors.Is(err, datasource.ErrCrawlConfigRequired):
		return errCrawlConfigRequired
	case errors.Is(err, datasource.ErrCreateFailed):
		return errCreateFailed
	case errors.Is(err, datasource.ErrUpdateFailed):
		return errUpdateFailed
	case errors.Is(err, datasource.ErrDeleteFailed):
		return errDeleteFailed
	case errors.Is(err, datasource.ErrListFailed):
		return errListFailed
	case errors.Is(err, datasource.ErrUpdateNotAllowed):
		return errUpdateNotAllowed
	case errors.Is(err, datasource.ErrInvalidTransition):
		return errInvalidTransition
	case errors.Is(err, datasource.ErrActivateNotAllowed):
		return errActivateNotAllowed
	case errors.Is(err, datasource.ErrPauseNotAllowed):
		return errPauseNotAllowed
	case errors.Is(err, datasource.ErrResumeNotAllowed):
		return errResumeNotAllowed
	case errors.Is(err, datasource.ErrCrawlModeNotAllowed):
		return errCrawlModeNotAllowed
	case errors.Is(err, datasource.ErrTargetNotFound):
		return errTargetNotFound
	case errors.Is(err, datasource.ErrTargetValuesRequired):
		return errTargetValuesRequired
	case errors.Is(err, datasource.ErrTargetValuesMustBeURLs):
		return errTargetValuesMustBeURLs
	case errors.Is(err, datasource.ErrInvalidTargetType):
		return errInvalidTargetType
	case errors.Is(err, datasource.ErrSourceNotCrawl):
		return errSourceNotCrawl
	case errors.Is(err, datasource.ErrTargetCreateFailed):
		return errTargetCreateFailed
	case errors.Is(err, datasource.ErrTargetUpdateFailed):
		return errTargetUpdateFailed
	case errors.Is(err, datasource.ErrTargetDeleteFailed):
		return errTargetDeleteFailed
	case errors.Is(err, datasource.ErrTargetListFailed):
		return errTargetListFailed
	case errors.Is(err, datasource.ErrInvalidTargetInterval):
		return errInvalidTargetInterval
	default:
		return errInternal
	}
}

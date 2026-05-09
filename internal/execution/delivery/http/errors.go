package http

import (
	"errors"
	"net/http"

	"ingest-srv/internal/execution"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errDatasourceNotFound = &pkgErrors.HTTPError{Code: 1, Message: "Data source not found", StatusCode: http.StatusNotFound}
	errTargetNotFound     = &pkgErrors.HTTPError{Code: 2, Message: "Crawl target not found", StatusCode: http.StatusNotFound}
	errDispatchNotAllowed = &pkgErrors.HTTPError{Code: 3, Message: "Dispatch is not allowed for this target", StatusCode: http.StatusConflict}
	errUnsupportedMapping = &pkgErrors.HTTPError{Code: 4, Message: "Unsupported dispatch mapping", StatusCode: http.StatusBadRequest}
	errParseIDsRequired   = &pkgErrors.HTTPError{Code: 5, Message: "facebook post target requires platform_meta.parse_ids", StatusCode: http.StatusBadRequest}
	errWrongPath          = &pkgErrors.HTTPError{Code: 6, Message: "Wrong request path", StatusCode: http.StatusBadRequest}
	errFacebookPageID     = &pkgErrors.HTTPError{Code: 7, Message: "Facebook profile target requires numeric page_id", StatusCode: http.StatusBadRequest}
	errTikTokProfile      = &pkgErrors.HTTPError{Code: 8, Message: "TikTok profile target requires username or sec_uid", StatusCode: http.StatusBadRequest}
	errDispatchFailed     = &pkgErrors.HTTPError{Code: 99, Message: "Failed to dispatch execution task", StatusCode: http.StatusInternalServerError}
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, execution.ErrDataSourceNotFound):
		return errDatasourceNotFound
	case errors.Is(err, execution.ErrTargetNotFound):
		return errTargetNotFound
	case errors.Is(err, execution.ErrDispatchNotAllowed):
		return errDispatchNotAllowed
	case errors.Is(err, execution.ErrUnsupportedDispatchMapping):
		return errUnsupportedMapping
	case errors.Is(err, execution.ErrPlatformMetaParseIDs):
		return errParseIDsRequired
	case errors.Is(err, execution.ErrFacebookPageIDRequired):
		return errFacebookPageID
	case errors.Is(err, execution.ErrTikTokProfileRequired):
		return errTikTokProfile
	default:
		return errDispatchFailed
	}
}

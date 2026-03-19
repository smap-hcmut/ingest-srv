package http

import (
	"errors"
	"ingest-srv/internal/dryrun"
	"net/http"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errSourceNotFound     = &pkgErrors.HTTPError{Code: 1, Message: "Datasource not found", StatusCode: http.StatusBadRequest}
	errTargetNotFound     = &pkgErrors.HTTPError{Code: 2, Message: "Crawl target not found", StatusCode: http.StatusBadRequest}
	errTargetRequired     = &pkgErrors.HTTPError{Code: 3, Message: "target_id is required for crawl dryrun", StatusCode: http.StatusBadRequest}
	errTargetForbidden    = &pkgErrors.HTTPError{Code: 4, Message: "target_id is not allowed for passive dryrun", StatusCode: http.StatusBadRequest}
	errDryrunNotAllowed   = &pkgErrors.HTTPError{Code: 5, Message: "Dryrun is not allowed in current source state", StatusCode: http.StatusBadRequest}
	errInvalidSampleLimit = &pkgErrors.HTTPError{Code: 6, Message: "sample_limit must be greater than 0", StatusCode: http.StatusBadRequest}
	errResultNotFound     = &pkgErrors.HTTPError{Code: 7, Message: "Dryrun result not found", StatusCode: http.StatusBadRequest}
	errCreateFailed       = &pkgErrors.HTTPError{Code: 8, Message: "Failed to create dryrun result", StatusCode: http.StatusInternalServerError}
	errGetFailed          = &pkgErrors.HTTPError{Code: 9, Message: "Failed to get dryrun result", StatusCode: http.StatusInternalServerError}
	errUpdateFailed       = &pkgErrors.HTTPError{Code: 10, Message: "Failed to update dryrun result", StatusCode: http.StatusInternalServerError}
	errListFailed         = &pkgErrors.HTTPError{Code: 11, Message: "Failed to list dryrun history", StatusCode: http.StatusInternalServerError}
	errWrongBody          = &pkgErrors.HTTPError{Code: 12, Message: "Wrong request body", StatusCode: http.StatusBadRequest}
	errUnsupportedMapping = &pkgErrors.HTTPError{Code: 13, Message: "Dryrun mapping is not supported yet", StatusCode: http.StatusBadRequest}
	errDispatchFailed     = &pkgErrors.HTTPError{Code: 14, Message: "Failed to dispatch dryrun task", StatusCode: http.StatusInternalServerError}
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, dryrun.ErrSourceNotFound):
		return errSourceNotFound
	case errors.Is(err, dryrun.ErrTargetNotFound):
		return errTargetNotFound
	case errors.Is(err, dryrun.ErrTargetRequired):
		return errTargetRequired
	case errors.Is(err, dryrun.ErrTargetForbidden):
		return errTargetForbidden
	case errors.Is(err, dryrun.ErrDryrunNotAllowed):
		return errDryrunNotAllowed
	case errors.Is(err, dryrun.ErrUnsupportedMapping):
		return errUnsupportedMapping
	case errors.Is(err, dryrun.ErrInvalidSampleLimit):
		return errInvalidSampleLimit
	case errors.Is(err, dryrun.ErrResultNotFound):
		return errResultNotFound
	case errors.Is(err, dryrun.ErrDispatchFailed):
		return errDispatchFailed
	case errors.Is(err, dryrun.ErrCreateFailed):
		return errCreateFailed
	case errors.Is(err, dryrun.ErrGetFailed):
		return errGetFailed
	case errors.Is(err, dryrun.ErrUpdateFailed):
		return errUpdateFailed
	case errors.Is(err, dryrun.ErrListFailed):
		return errListFailed
	default:
		return err
	}
}

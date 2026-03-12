package http

import (
	"ingest-srv/internal/dryrun"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errSourceNotFound     = pkgErrors.NewHTTPError(171001, "Datasource not found")
	errTargetNotFound     = pkgErrors.NewHTTPError(171002, "Crawl target not found")
	errTargetRequired     = pkgErrors.NewHTTPError(171003, "target_id is required for crawl dryrun")
	errTargetForbidden    = pkgErrors.NewHTTPError(171004, "target_id is not allowed for passive dryrun")
	errDryrunNotAllowed   = pkgErrors.NewHTTPError(171005, "Dryrun is not allowed in current source state")
	errInvalidSampleLimit = pkgErrors.NewHTTPError(171006, "sample_limit must be greater than 0")
	errResultNotFound     = pkgErrors.NewHTTPError(171007, "Dryrun result not found")
	errCreateFailed       = pkgErrors.NewHTTPError(171008, "Failed to create dryrun result")
	errGetFailed          = pkgErrors.NewHTTPError(171009, "Failed to get dryrun result")
	errUpdateFailed       = pkgErrors.NewHTTPError(171010, "Failed to update dryrun result")
	errListFailed         = pkgErrors.NewHTTPError(171011, "Failed to list dryrun history")
	errWrongBody          = pkgErrors.NewHTTPError(171012, "Wrong request body")
)

func (h *handler) mapError(err error) error {
	switch err {
	case dryrun.ErrSourceNotFound:
		return errSourceNotFound
	case dryrun.ErrTargetNotFound:
		return errTargetNotFound
	case dryrun.ErrTargetRequired:
		return errTargetRequired
	case dryrun.ErrTargetForbidden:
		return errTargetForbidden
	case dryrun.ErrDryrunNotAllowed:
		return errDryrunNotAllowed
	case dryrun.ErrInvalidSampleLimit:
		return errInvalidSampleLimit
	case dryrun.ErrResultNotFound:
		return errResultNotFound
	case dryrun.ErrCreateFailed:
		return errCreateFailed
	case dryrun.ErrGetFailed:
		return errGetFailed
	case dryrun.ErrUpdateFailed:
		return errUpdateFailed
	case dryrun.ErrListFailed:
		return errListFailed
	default:
		return err
	}
}

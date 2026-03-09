package http

import (
	"ingest-srv/internal/execution"
	pkgErrors "ingest-srv/pkg/errors"
)

var (
	errDatasourceNotFound = &pkgErrors.HTTPError{Code: 172001, Message: "Data source not found", StatusCode: 404}
	errTargetNotFound     = &pkgErrors.HTTPError{Code: 172002, Message: "Crawl target not found", StatusCode: 404}
	errDispatchNotAllowed = pkgErrors.NewHTTPError(172003, "Dispatch is not allowed for this target")
	errUnsupportedMapping = pkgErrors.NewHTTPError(172004, "Unsupported dispatch mapping")
	errParseIDsRequired   = pkgErrors.NewHTTPError(172005, "facebook post target requires platform_meta.parse_ids")
	errWrongPath          = pkgErrors.NewHTTPError(172006, "Wrong request path")
	errDispatchFailed     = &pkgErrors.HTTPError{Code: 172999, Message: "Failed to dispatch execution task", StatusCode: 500}
)

func (h *handler) mapError(err error) error {
	switch err {
	case execution.ErrDataSourceNotFound:
		return errDatasourceNotFound
	case execution.ErrTargetNotFound:
		return errTargetNotFound
	case execution.ErrDispatchNotAllowed:
		return errDispatchNotAllowed
	case execution.ErrUnsupportedDispatchMapping:
		return errUnsupportedMapping
	case execution.ErrPlatformMetaParseIDs:
		return errParseIDsRequired
	default:
		return errDispatchFailed
	}
}

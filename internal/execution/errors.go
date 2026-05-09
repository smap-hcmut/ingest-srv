package execution

import "errors"

// Domain errors for execution runtime.
var (
	ErrDataSourceNotFound         = errors.New("execution datasource not found")
	ErrTargetNotFound             = errors.New("execution target not found")
	ErrDispatchNotAllowed         = errors.New("dispatch not allowed for target")
	ErrUnsupportedDispatchMapping = errors.New("unsupported dispatch mapping")
	ErrPlatformMetaParseIDs       = errors.New("facebook post target requires platform_meta.parse_ids")
	ErrFacebookPageIDRequired     = errors.New("facebook profile target requires numeric page_id in platform_meta or URL")
	ErrTikTokProfileRequired      = errors.New("tiktok profile target requires username or sec_uid")
	ErrDispatchFailed             = errors.New("failed to dispatch external task")
	ErrCancelRuntimeFailed        = errors.New("failed to cancel project runtime")
	ErrCompletionTaskNotFound     = errors.New("execution completion task not found")
	ErrInvalidCompletionInput     = errors.New("invalid completion input")
)

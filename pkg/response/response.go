package response

import (
	"fmt"
	"net/http"
	"runtime"

	"smap-collector/pkg/discord"
	"smap-collector/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Resp is the response format.
type Resp struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Errors    any    `json:"errors,omitempty"`
}

// NewOKResp returns a new OK response with the given data.
func NewOKResp(data any) Resp {
	return Resp{
		ErrorCode: 0,
		Message:   "Success",
		Data:      data,
	}
}

// Ok returns a new OK response with the given data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, NewOKResp(data))
}

// Unauthorized returns a new Unauthorized response with the given data.
func Unauthorized(c *gin.Context) {
	c.JSON(parseError(errors.NewUnauthorizedHTTPError(), c, nil))
}

func Forbidden(c *gin.Context) {
	c.JSON(parseError(errors.NewForbiddenHTTPError(), c, nil))
}

func parseError(err error, c *gin.Context, d *discord.Discord) (int, Resp) {
	switch parsedErr := err.(type) {
	case *errors.ValidationError:
		return http.StatusBadRequest, Resp{
			ErrorCode: parsedErr.Code,
			Message:   parsedErr.Error(),
		}
	case *errors.PermissionError:
		return http.StatusBadRequest, Resp{
			ErrorCode: parsedErr.Code,
			Message:   parsedErr.Error(),
		}
	case *errors.ValidationErrorCollector:
		return http.StatusBadRequest, Resp{
			ErrorCode: ValidationErrorCode,
			Message:   ValidationErrorMsg,
			Errors:    parsedErr.Errors(),
		}
	case *errors.PermissionErrorCollector:
		return http.StatusBadRequest, Resp{
			ErrorCode: PermissionErrorCode,
			Message:   PermissionErrorMsg,
			Errors:    parsedErr.Errors(),
		}
	case *errors.HTTPError:
		statusCode := parsedErr.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusBadRequest
		}

		return statusCode, Resp{
			ErrorCode: parsedErr.Code,
			Message:   parsedErr.Message,
		}
	default:
		if d != nil {
			stackTrace := captureStackTrace()
			sendDiscordMessageAsync(c, d, buildInternalServerErrorDataForReportBug(c, err.Error(), stackTrace))
		}

		return http.StatusInternalServerError, Resp{
			ErrorCode: 500,
			Message:   DefaultErrorMessage,
		}
	}
}

// Error returns a new Error response with the given error.
func Error(c *gin.Context, err error, d *discord.Discord) {
	statusCode, resp := parseError(err, c, d)
	c.JSON(statusCode, resp)
}

// HttpError returns a new Error response with the given HTTP error.
func HttpError(c *gin.Context, err *errors.HTTPError) {
	statusCode, resp := parseError(err, c, nil)
	c.JSON(statusCode, resp)
}

// ErrorMapping is a map of error to HTTPError.
type ErrorMapping map[error]*errors.HTTPError

// ErrorWithMap returns a new Error response with the given error.
func ErrorWithMap(c *gin.Context, err error, eMap ErrorMapping) {
	if httpErr, ok := eMap[err]; ok {
		Error(c, httpErr, nil)
		return
	}

	Error(c, err, nil)
}

// PanicError handles panic errors and returns an error response.
func PanicError(c *gin.Context, err any, d *discord.Discord) {
	if err == nil {
		statusCode, resp := parseError(nil, c, nil)
		c.JSON(statusCode, resp)
	} else {
		if errVal, ok := err.(error); ok {
			statusCode, resp := parseError(errVal, c, d)
			c.JSON(statusCode, resp)
		} else {
			statusCode, resp := parseError(fmt.Errorf("%v", err), c, d)
			c.JSON(statusCode, resp)
		}
	}
}

// captureStackTrace captures the current stack trace.
func captureStackTrace() []string {
	var pcs [defaultStackTraceDepth]uintptr
	n := runtime.Callers(2, pcs[:])
	if n == 0 {
		return nil
	}

	var stackTrace []string
	for _, pc := range pcs[:n] {
		f := runtime.FuncForPC(pc)
		if f != nil {
			file, line := f.FileLine(pc)
			stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", file, line, f.Name()))
		}
	}

	return stackTrace
}

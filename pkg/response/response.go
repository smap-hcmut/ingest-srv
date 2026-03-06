package response

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strings"

	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/errors"

	"github.com/gin-gonic/gin"
)

// NewOKResp returns a new OK response with the given data.
func NewOKResp(data any) Resp {
	return Resp{
		ErrorCode: 0,
		Message:   MessageSuccess,
		Data:      data,
	}
}

// OK sends 200 JSON with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, NewOKResp(data))
}

// Unauthorized sends 401 response.
func Unauthorized(c *gin.Context) {
	c.JSON(parseError(errors.NewUnauthorizedHTTPError(), c, nil))
}

// Forbidden sends 403 response.
func Forbidden(c *gin.Context) {
	c.JSON(parseError(errors.NewForbiddenHTTPError(), c, nil))
}

func parseError(err error, c *gin.Context, d discord.IDiscord) (int, Resp) {
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
			ErrorCode: InternalServerErrorCode,
			Message:   DefaultErrorMessage,
		}
	}
}

// Error sends error response (status + JSON from parseError).
func Error(c *gin.Context, err error, d discord.IDiscord) {
	statusCode, resp := parseError(err, c, d)
	c.JSON(statusCode, resp)
}

// HttpError sends response for *errors.HTTPError.
func HttpError(c *gin.Context, err *errors.HTTPError) {
	statusCode, resp := parseError(err, c, nil)
	c.JSON(statusCode, resp)
}

// ErrorWithMap looks up err in eMap and sends corresponding HTTPError, else Error.
func ErrorWithMap(c *gin.Context, err error, eMap ErrorMapping) {
	if httpErr, ok := eMap[err]; ok {
		Error(c, httpErr, nil)
		return
	}
	Error(c, err, nil)
}

// PanicError handles panic recovery and sends error response.
func PanicError(c *gin.Context, err any, d discord.IDiscord) {
	if err == nil {
		statusCode, resp := parseError(nil, c, nil)
		c.JSON(statusCode, resp)
		return
	}
	if errVal, ok := err.(error); ok {
		statusCode, resp := parseError(errVal, c, d)
		c.JSON(statusCode, resp)
	} else {
		statusCode, resp := parseError(fmt.Errorf("%v", err), c, d)
		c.JSON(statusCode, resp)
	}
}

func captureStackTrace() []string {
	var pcs [DefaultStackTraceDepth]uintptr
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

func sendDiscordMessageAsync(c *gin.Context, d discord.IDiscord, message string) {
	if d == nil {
		return
	}
	go func() {
		for _, msg := range splitMessageForDiscord(message) {
			if err := d.ReportBug(context.Background(), msg); err != nil {
				log.Printf("pkg.response.sendDiscordMessageAsync.ReportBug: %v\n", err)
			}
		}
	}()
}

func splitMessageForDiscord(message string) []string {
	var chunks []string
	var current string
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		line += "\n"
		if len(current)+len(line) > DiscordMaxMessageLen {
			if current != "" {
				chunks = append(chunks, strings.TrimSuffix(current, "\n"))
				current = ""
			}
			for len(line) > DiscordMaxMessageLen {
				chunks = append(chunks, line[:DiscordMaxMessageLen])
				line = line[DiscordMaxMessageLen:]
			}
		}
		current += line
	}
	if current != "" {
		chunks = append(chunks, strings.TrimSuffix(current, "\n"))
	}
	return chunks
}

func buildInternalServerErrorDataForReportBug(c *gin.Context, errString string, backtrace []string) string {
	url := c.Request.URL.String()
	method := c.Request.Method
	params := c.Request.URL.Query().Encode()
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	body := string(bodyBytes)
	var sb strings.Builder
	sb.WriteString("================ SMAP SERVICE ERROR ================\n")
	sb.WriteString(fmt.Sprintf("Route   : %s\n", url))
	sb.WriteString(fmt.Sprintf("Method  : %s\n", method))
	sb.WriteString("----------------------------------------------------\n")
	if len(c.Request.Header) > 0 {
		sb.WriteString("Headers :\n")
		for key, values := range c.Request.Header {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", key, strings.Join(values, ", ")))
		}
		sb.WriteString("----------------------------------------------------\n")
	}
	if params != "" {
		sb.WriteString(fmt.Sprintf("Params  : %s\n", params))
	}
	if body != "" {
		sb.WriteString("Body    :\n")
		var prettyBody bytes.Buffer
		if err := json.Indent(&prettyBody, bodyBytes, "    ", "  "); err == nil {
			sb.WriteString(prettyBody.String() + "\n")
		} else {
			sb.WriteString("    " + body + "\n")
		}
		sb.WriteString("----------------------------------------------------\n")
	}
	sb.WriteString(fmt.Sprintf("Error   : %s\n", errString))
	if len(backtrace) > 0 {
		sb.WriteString("\nBacktrace:\n")
		for i, line := range backtrace {
			sb.WriteString(fmt.Sprintf("[%d]: %s\n", i, line))
		}
	}
	sb.WriteString("====================================================\n")
	return sb.String()
}

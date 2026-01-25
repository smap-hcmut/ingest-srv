package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"smap-collector/pkg/discord"

	"github.com/gin-gonic/gin"
)

// sendDiscordMessageAsync sends Discord messages asynchronously.
func sendDiscordMessageAsync(c *gin.Context, d *discord.Discord, message string) {
	if d == nil {
		return
	}

	go func() {
		splitMessages := splitMessageForDiscord(message)
		for _, msg := range splitMessages {
			err := d.ReportBug(c.Request.Context(), msg)
			if err != nil {
				// Use standard log as fallback since we're in async goroutine
				log.Printf("pkg.response.sendDiscordMessageAsync.ReportBug: %v\n", err)
			}
		}
	}()
}

// splitMessageForDiscord splits a message into chunks that fit Discord's message length limits.
func splitMessageForDiscord(message string) []string {
	const maxLen = 5000
	var chunks []string
	var current string
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		line += "\n"
		if len(current)+len(line) > maxLen {
			if current != "" {
				chunks = append(chunks, strings.TrimSuffix(current, "\n"))
				current = ""
			}
			for len(line) > maxLen {
				chunks = append(chunks, line[:maxLen])
				line = line[maxLen:]
			}
		}
		current += line
	}
	if current != "" {
		chunks = append(chunks, strings.TrimSuffix(current, "\n"))
	}
	return chunks
}

// buildInternalServerErrorDataForReportBug builds a formatted error report for Discord.
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
		// Pretty print JSON if possible
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

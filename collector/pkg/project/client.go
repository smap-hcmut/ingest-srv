package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (c *httpClient) SendDryRunCallback(ctx context.Context, req CallbackRequest) error {
	delay := c.initialDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		err := c.doSend(ctx, req, attempt+1)
		if err == nil {
			return nil
		}

		if attempt < c.maxRetries {
			c.l.Warnf(ctx, "Webhook attempt %d/%d failed, retrying in %v: %v",
				attempt+1, c.maxRetries+1, delay, err)

			time.Sleep(delay)

			// Exponential backoff
			delay *= 2
			if delay > c.maxDelay {
				delay = c.maxDelay
			}
		} else {
			c.l.Errorf(ctx, "Webhook failed after %d attempts: %v", c.maxRetries+1, err)
		}
	}

	return fmt.Errorf("webhook failed after %d attempts", c.maxRetries+1)
}

func (c *httpClient) doSend(ctx context.Context, req CallbackRequest, attempt int) error {
	// Marshal request to JSON
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	url := c.baseURL + dryRunCallbackEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.internalKey)

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProjectUnavailable, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.l.Infof(ctx, "Webhook callback sent successfully: job_id=%s, platform=%s, status=%s, attempt=%d",
			req.JobID, req.Platform, req.Status, attempt)
		return nil
	}

	return c.handleErrorResponse(resp)
}

func (c *httpClient) SendProgressCallback(ctx context.Context, req ProgressCallbackRequest) error {
	delay := c.initialDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		err := c.doSendProgress(ctx, req, attempt+1)
		if err == nil {
			return nil
		}

		if attempt < c.maxRetries {
			c.l.Warnf(ctx, "Progress webhook attempt %d/%d failed, retrying in %v: %v",
				attempt+1, c.maxRetries+1, delay, err)

			time.Sleep(delay)

			delay *= 2
			if delay > c.maxDelay {
				delay = c.maxDelay
			}
		} else {
			c.l.Errorf(ctx, "Progress webhook failed after %d attempts: %v", c.maxRetries+1, err)
		}
	}

	return fmt.Errorf("progress webhook failed after %d attempts", c.maxRetries+1)
}

func (c *httpClient) doSendProgress(ctx context.Context, req ProgressCallbackRequest, attempt int) error {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	url := c.baseURL + progressCallbackEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Key", c.internalKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProjectUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.l.Infof(ctx, "Progress callback sent: project_id=%s, status=%s, attempt=%d",
			req.ProjectID, req.Status, attempt)
		return nil
	}

	return c.handleErrorResponse(resp)
}

func (c *httpClient) handleErrorResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrProjectUnauthorized
	case http.StatusRequestTimeout:
		return ErrProjectTimeout
	case http.StatusBadRequest:
		return fmt.Errorf("webhook returned client error %d (not retrying)", resp.StatusCode)
	default:
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return fmt.Errorf("webhook returned client error %d (not retrying)", resp.StatusCode)
		}
		return fmt.Errorf("%w: status %d", ErrProjectUnavailable, resp.StatusCode)
	}
}

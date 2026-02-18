package http_util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SisyphusSQ/summary-sys/utils/retry"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultRetries   = 3
	defaultRetryWait = 1 * time.Second
	maxErrorBodySize = 256
	maxResponseSize  = 2 * 1024 * 1024
)

// Client wraps http.Client with retry support.
type Client struct {
	httpClient *http.Client
	retries    int
	retryWait  time.Duration
}

// NewClient creates an HTTP client with timeout and retry defaults.
func NewClient(timeout time.Duration, retries int, retryWait time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if retries <= 0 {
		retries = defaultRetries
	}
	if retryWait <= 0 {
		retryWait = defaultRetryWait
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		retries:    retries,
		retryWait:  retryWait,
	}
}

// Do executes the request with retry for transient network errors and statuses.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if err := prepareReplayableBody(req); err != nil {
		return nil, err
	}

	var response *http.Response
	err := retry.DoWithContext(req.Context(), func() error {
		attemptReq, err := cloneRequest(req)
		if err != nil {
			return err
		}

		response, err = c.httpClient.Do(attemptReq)
		if err != nil {
			return err
		}
		if shouldRetryStatus(response.StatusCode) {
			_ = response.Body.Close()
			return fmt.Errorf("transient http status: %d", response.StatusCode)
		}

		return nil
	}, c.retries, c.retryWait)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DoJSON sends a JSON request and optionally unmarshals JSON response.
func (c *Client) DoJSON(
	ctx context.Context,
	method string,
	url string,
	headers map[string]string,
	requestBody any,
	responseBody any,
) (*http.Response, error) {
	var bodyReader io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	response, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return response, fmt.Errorf("read response body: %w", err)
	}
	_ = response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(payload))
	if len(payload) > maxResponseSize {
		return response, fmt.Errorf(
			"response body too large: exceeds %d bytes",
			maxResponseSize,
		)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return response, fmt.Errorf(
			"http request failed: status=%d body=%s",
			response.StatusCode,
			compactBody(payload),
		)
	}

	if responseBody != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, responseBody); err != nil {
			return response, fmt.Errorf("unmarshal response body: %w", err)
		}
	}

	return response, nil
}

func prepareReplayableBody(req *http.Request) error {
	if req.Body == nil || req.GetBody != nil {
		return nil
	}

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	_ = req.Body.Close()
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	req.Body = io.NopCloser(bytes.NewReader(payload))

	return nil
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	cloned := req.Clone(req.Context())
	if req.GetBody == nil {
		return cloned, nil
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("clone request body: %w", err)
	}
	cloned.Body = body

	return cloned, nil
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func compactBody(payload []byte) string {
	body := strings.TrimSpace(string(payload))
	if body == "" {
		return "<empty>"
	}
	if len(body) > maxErrorBodySize {
		return body[:maxErrorBodySize] + "..."
	}

	return body
}

package exa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.exa.ai"
	maxAttempts    = 3
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	RetryDelay time.Duration
}

type Response struct {
	Raw json.RawMessage
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("exa api returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("exa api returned HTTP %d: %s", e.StatusCode, body)
}

func (c Client) Search(ctx context.Context, body map[string]any) (Response, error) {
	return c.post(ctx, "/search", body)
}

func (c Client) Contents(ctx context.Context, body map[string]any) (Response, error) {
	return c.post(ctx, "/contents", body)
}

func (c Client) post(ctx context.Context, path string, body map[string]any) (Response, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return Response{}, errors.New("missing Exa API key")
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, fmt.Errorf("encode request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return Response{}, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-api-key", c.APIKey)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			if canRetry(attempt) && retryableNetworkError(ctx, err) {
				sleep(ctx, c.RetryDelay)
				continue
			}
			return Response{}, fmt.Errorf("send request: %w", err)
		}

		data, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return Response{}, fmt.Errorf("read response: %w", readErr)
		}
		if closeErr != nil {
			return Response{}, fmt.Errorf("close response: %w", closeErr)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return Response{Raw: json.RawMessage(data)}, nil
		}

		apiErr := APIError{StatusCode: resp.StatusCode, Body: string(data)}
		if canRetry(attempt) && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) {
			lastErr = apiErr
			sleep(ctx, c.RetryDelay)
			continue
		}
		return Response{}, apiErr
	}
	return Response{}, lastErr
}

func canRetry(attempt int) bool {
	return attempt < maxAttempts-1
}

func retryableNetworkError(ctx context.Context, err error) bool {
	return ctx.Err() == nil && err != nil
}

func sleep(ctx context.Context, delay time.Duration) {
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

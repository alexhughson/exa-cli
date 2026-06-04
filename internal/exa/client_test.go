package exa

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestSearchPostsJSONWithAPIKey(t *testing.T) {
	var sawPath, sawKey string
	var sawBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sawPath = r.URL.Path
		sawKey = r.Header.Get("x-api-key")
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return jsonResponse(http.StatusOK, `{"results":[{"title":"A","url":"https://example.com"}]}`), nil
	})}

	resp, err := Client{BaseURL: "https://api.test", APIKey: "test-key", HTTPClient: httpClient}.Search(context.Background(), map[string]any{
		"query": "hello",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if sawPath != "/search" {
		t.Fatalf("path = %q", sawPath)
	}
	if sawKey != "test-key" {
		t.Fatalf("api key header = %q", sawKey)
	}
	if sawBody["query"] != "hello" {
		t.Fatalf("query = %#v", sawBody["query"])
	}
	if string(resp.Raw) == "" {
		t.Fatal("empty raw response")
	}
}

func TestContentsRetriesRateLimit(t *testing.T) {
	attempts := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return jsonResponse(http.StatusTooManyRequests, `rate limited`), nil
		}
		return jsonResponse(http.StatusOK, `{"results":[]}`), nil
	})}

	_, err := Client{BaseURL: "https://api.test", APIKey: "test-key", HTTPClient: httpClient}.Contents(context.Background(), map[string]any{
		"urls": []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("Contents() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestMissingAPIKey(t *testing.T) {
	_, err := Client{}.Search(context.Background(), map[string]any{"query": "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

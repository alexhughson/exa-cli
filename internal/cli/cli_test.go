package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCommandSendsRequest(t *testing.T) {
	var sawBody map[string]any
	var sawKey string
	withHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		sawKey = r.Header.Get("x-api-key")
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return jsonResponse(http.StatusOK, `{"results":[{"title":"Result","url":"https://example.com","summary":"A summary"}]}`), nil
	})})

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"--json", "--api-base", "https://api.test", "search", "--num-results", "2", "machine learning"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{"EXA_API_KEY": "env-key"}),
		"test",
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if sawKey != "env-key" {
		t.Fatalf("api key = %q", sawKey)
	}
	if sawBody["query"] != "machine learning" {
		t.Fatalf("query = %#v", sawBody["query"])
	}
	if sawBody["numResults"].(float64) != 2 {
		t.Fatalf("numResults = %#v", sawBody["numResults"])
	}
	if !strings.Contains(stdout.String(), `"Result"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestFetchCommandSendsContentsRequest(t *testing.T) {
	var sawBody map[string]any
	withHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/contents" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return jsonResponse(http.StatusOK, `{"results":[{"url":"https://example.com","text":"Body"}]}`), nil
	})})

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"--api-base", "https://api.test", "fetch", "--max-characters", "42", "https://example.com"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{"EXA_API_KEY": "env-key"}),
		"test",
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	urls := sawBody["urls"].([]any)
	if urls[0] != "https://example.com" {
		t.Fatalf("urls = %#v", urls)
	}
	text := sawBody["text"].(map[string]any)
	if text["maxCharacters"].(float64) != 42 {
		t.Fatalf("text = %#v", text)
	}
	if !strings.Contains(stdout.String(), "Body") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestSearchCommandUsesFreeTierWithoutAPIKey(t *testing.T) {
	var sawKey string
	withHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sawKey = r.Header.Get("x-api-key")
		return jsonResponse(http.StatusOK, `{"results":[{"title":"Free Tier","url":"https://example.com"}]}`), nil
	})})

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"--api-base", "https://api.test", "search", "machine learning"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{}),
		"test",
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if sawKey != "" {
		t.Fatalf("api key = %q, want empty", sawKey)
	}
	if !strings.Contains(stdout.String(), "Free Tier") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestSearchCommandReportsLoginWhenFreeTierFails(t *testing.T) {
	withHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"error":"missing api key"}`), nil
	})})

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"--api-base", "https://api.test", "search", "machine learning"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{}),
		"test",
	)
	if code != 1 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Exa free-tier request failed:") {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Run `exa-cli login` to configure an API key.") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestAdvancedSearchCommandSendsFilters(t *testing.T) {
	var sawBody map[string]any
	withHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return jsonResponse(http.StatusOK, `{"results":[]}`), nil
	})})

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{
			"--api-base", "https://api.test",
			"advanced-search",
			"--type", "deep",
			"--category", "research paper",
			"--include-domain", "arxiv.org,openreview.net",
			"--summary",
			"--text-max-characters", "1000",
			"agent benchmarks",
		},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{"EXA_API_KEY": "env-key"}),
		"test",
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if sawBody["type"] != "deep" {
		t.Fatalf("type = %#v", sawBody["type"])
	}
	domains := sawBody["includeDomains"].([]any)
	if len(domains) != 2 {
		t.Fatalf("domains = %#v", domains)
	}
	contents := sawBody["contents"].(map[string]any)
	if contents["summary"] != true {
		t.Fatalf("contents = %#v", contents)
	}
	text := contents["text"].(map[string]any)
	if text["maxCharacters"].(float64) != 1000 {
		t.Fatalf("text = %#v", text)
	}
}

func TestAuthCommandWritesConfig(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"auth", "exa-key"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{"HOME": home}),
		"test",
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join(home, ".exa-cli", "config.json")) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestLoginCommandWritesConfig(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"login", "exa-key"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		testLookup(map[string]string{"HOME": home}),
		"test",
	)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join(home, ".exa-cli", "config.json")) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func testLookup(values map[string]string) envLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func withHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()
	original := httpClient
	httpClient = client
	t.Cleanup(func() {
		httpClient = original
	})
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

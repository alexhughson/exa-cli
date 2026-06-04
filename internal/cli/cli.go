package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alex/exa-cli/internal/config"
	"github.com/alex/exa-cli/internal/exa"
	outfmt "github.com/alex/exa-cli/internal/format"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*s = append(*s, part)
		}
	}
	return nil
}

type envLookup func(string) (string, bool)

var httpClient = http.DefaultClient

type globals struct {
	json    bool
	timeout time.Duration
	baseURL string
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, lookup envLookup, version string) int {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	global, rest, err := parseGlobals(args, lookup)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if len(rest) == 0 {
		usage(stdout, version)
		return 0
	}

	cmd, cmdArgs := normalize(rest[0]), rest[1:]
	switch cmd {
	case "help", "-h", "--help":
		usage(stdout, version)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintln(stdout, version)
		return 0
	case "auth":
		return runAuth(cmdArgs, stdin, stdout, stderr, lookup)
	case "search", "web-search", "web_search_exa":
		return runSearch(cmdArgs, stdout, stderr, lookup, global)
	case "fetch", "web-fetch", "web_fetch_exa":
		return runFetch(cmdArgs, stdout, stderr, lookup, global)
	case "advanced-search", "web-search-advanced", "web_search_advanced_exa":
		return runAdvancedSearch(cmdArgs, stdout, stderr, lookup, global)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", rest[0])
		usage(stderr, version)
		return 2
	}
}

func parseGlobals(args []string, lookup envLookup) (globals, []string, error) {
	g := globals{timeout: 60 * time.Second}
	if base, ok := lookup("EXA_API_BASE"); ok && strings.TrimSpace(base) != "" {
		g.baseURL = strings.TrimSpace(base)
	}

	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			g.json = true
		case "--timeout":
			i++
			if i >= len(args) {
				return g, nil, errors.New("--timeout requires a value")
			}
			timeout, err := time.ParseDuration(args[i])
			if err != nil {
				return g, nil, fmt.Errorf("invalid --timeout: %w", err)
			}
			g.timeout = timeout
		case "--api-base":
			i++
			if i >= len(args) {
				return g, nil, errors.New("--api-base requires a value")
			}
			g.baseURL = args[i]
		default:
			rest = append(rest, args[i:]...)
			return g, rest, nil
		}
	}
	return g, rest, nil
}

func runAuth(args []string, stdin io.Reader, stdout, stderr io.Writer, lookup envLookup) int {
	if len(args) > 0 {
		switch normalize(args[0]) {
		case "status":
			source, err := config.LoadAPIKey(config.LookupEnv(lookup))
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if source.Key == "" {
				fmt.Fprintln(stdout, "Not authenticated")
				return 1
			}
			fmt.Fprintf(stdout, "Authenticated via %s (%s)\n", source.Source, config.Redact(source.Key))
			return 0
		case "logout":
			path, err := config.Logout(config.LookupEnv(lookup))
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			fmt.Fprintf(stdout, "Removed credentials from %s\n", path)
			return 0
		}
	}

	apiKey := ""
	if len(args) > 0 {
		apiKey = args[0]
	} else {
		var err error
		apiKey, err = readSecret("Exa API key: ", stdin, stderr)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	path, err := config.SaveAPIKey(apiKey, config.LookupEnv(lookup))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Saved credentials to %s\n", path)
	return 0
}

func runSearch(args []string, stdout, stderr io.Writer, lookup envLookup, global globals) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	numResults := fs.Int("num-results", 10, "number of results to return")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		fmt.Fprintln(stderr, "search requires a query")
		return 2
	}

	body := map[string]any{
		"query":      query,
		"numResults": *numResults,
		"contents": map[string]any{
			"text":       true,
			"highlights": true,
			"summary":    true,
		},
	}
	return runExa(stdout, stderr, lookup, global, func(ctx context.Context, client exa.Client) (exa.Response, error) {
		return client.Search(ctx, body)
	})
}

func runFetch(args []string, stdout, stderr io.Writer, lookup envLookup, global globals) int {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	maxCharacters := fs.Int("max-characters", 3000, "maximum characters per page")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	urls := fs.Args()
	if len(urls) == 0 {
		fmt.Fprintln(stderr, "fetch requires at least one URL")
		return 2
	}

	body := map[string]any{
		"urls": urls,
		"text": map[string]any{
			"maxCharacters": *maxCharacters,
		},
	}
	return runExa(stdout, stderr, lookup, global, func(ctx context.Context, client exa.Client) (exa.Response, error) {
		return client.Contents(ctx, body)
	})
}

func runAdvancedSearch(args []string, stdout, stderr io.Writer, lookup envLookup, global globals) int {
	fs := flag.NewFlagSet("advanced-search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	numResults := fs.Int("num-results", 10, "number of results to return")
	searchType := fs.String("type", "", "search type")
	category := fs.String("category", "", "search category")
	startCrawlDate := fs.String("start-crawl-date", "", "start crawl date")
	endCrawlDate := fs.String("end-crawl-date", "", "end crawl date")
	startPublishedDate := fs.String("start-published-date", "", "start published date")
	endPublishedDate := fs.String("end-published-date", "", "end published date")
	userLocation := fs.String("user-location", "", "user location")
	moderation := fs.Bool("moderation", false, "enable moderation")
	text := fs.Bool("text", true, "include text contents")
	noText := fs.Bool("no-text", false, "disable text contents")
	textMaxCharacters := fs.Int("text-max-characters", 0, "maximum text characters")
	highlights := fs.Bool("highlights", false, "include highlights")
	summary := fs.Bool("summary", false, "include summaries")
	summaryQuery := fs.String("summary-query", "", "summary query")
	contextFlag := fs.Bool("context", false, "include context")
	contextMaxCharacters := fs.Int("context-max-characters", 0, "maximum context characters")
	livecrawl := fs.String("livecrawl", "", "livecrawl mode")
	livecrawlTimeout := fs.Int("livecrawl-timeout", 0, "livecrawl timeout in milliseconds")
	subpages := fs.Int("subpages", 0, "number of subpages")
	subpageTarget := fs.String("subpage-target", "", "subpage target")
	maxAgeHours := fs.Int("max-age-hours", 0, "freshness window in hours")

	var includeDomains, excludeDomains, includeText, excludeText, includeURLs, excludeURLs, additionalQueries stringList
	fs.Var(&includeDomains, "include-domain", "domain to include; repeatable or comma-separated")
	fs.Var(&excludeDomains, "exclude-domain", "domain to exclude; repeatable or comma-separated")
	fs.Var(&includeText, "include-text", "text that must appear; repeatable or comma-separated")
	fs.Var(&excludeText, "exclude-text", "text that must not appear; repeatable or comma-separated")
	fs.Var(&includeURLs, "include-url", "URL to include; repeatable or comma-separated")
	fs.Var(&excludeURLs, "exclude-url", "URL to exclude; repeatable or comma-separated")
	fs.Var(&additionalQueries, "additional-query", "additional query; repeatable or comma-separated")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		fmt.Fprintln(stderr, "advanced-search requires a query")
		return 2
	}

	body := map[string]any{
		"query":      query,
		"numResults": *numResults,
	}
	putString(body, "type", *searchType)
	putString(body, "category", *category)
	putString(body, "startCrawlDate", *startCrawlDate)
	putString(body, "endCrawlDate", *endCrawlDate)
	putString(body, "startPublishedDate", *startPublishedDate)
	putString(body, "endPublishedDate", *endPublishedDate)
	putString(body, "userLocation", *userLocation)
	if *moderation {
		body["moderation"] = true
	}
	putList(body, "includeDomains", includeDomains)
	putList(body, "excludeDomains", excludeDomains)
	putList(body, "includeText", includeText)
	putList(body, "excludeText", excludeText)
	putList(body, "includeUrls", includeURLs)
	putList(body, "excludeUrls", excludeURLs)
	putList(body, "additionalQueries", additionalQueries)
	if *maxAgeHours > 0 {
		body["maxAgeHours"] = *maxAgeHours
	}

	contents := map[string]any{}
	if *noText {
		contents["text"] = false
	} else if *textMaxCharacters > 0 {
		contents["text"] = map[string]any{"maxCharacters": *textMaxCharacters}
	} else if *text {
		contents["text"] = true
	}
	if *highlights {
		contents["highlights"] = true
	}
	if *summaryQuery != "" {
		contents["summary"] = map[string]any{"query": *summaryQuery}
	} else if *summary {
		contents["summary"] = true
	}
	if *contextFlag {
		if *contextMaxCharacters > 0 {
			contents["context"] = map[string]any{"maxCharacters": *contextMaxCharacters}
		} else {
			contents["context"] = true
		}
	}
	putString(contents, "livecrawl", *livecrawl)
	if *livecrawlTimeout > 0 {
		contents["livecrawlTimeout"] = *livecrawlTimeout
	}
	if *subpages > 0 {
		contents["subpages"] = *subpages
	}
	putString(contents, "subpageTarget", *subpageTarget)
	if len(contents) > 0 {
		body["contents"] = contents
	}

	return runExa(stdout, stderr, lookup, global, func(ctx context.Context, client exa.Client) (exa.Response, error) {
		return client.Search(ctx, body)
	})
}

func runExa(stdout, stderr io.Writer, lookup envLookup, global globals, call func(context.Context, exa.Client) (exa.Response, error)) int {
	key, err := config.LoadAPIKey(config.LookupEnv(lookup))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if key.Key == "" {
		fmt.Fprintln(stderr, "missing Exa API key; set EXA_API_KEY or run `exa-cli auth <api-key>`")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), global.timeout)
	defer cancel()

	resp, err := call(ctx, exa.Client{
		BaseURL:    global.baseURL,
		APIKey:     key.Key,
		HTTPClient: httpClient,
		RetryDelay: 250 * time.Millisecond,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if global.json {
		err = outfmt.JSON(stdout, resp.Raw)
	} else {
		err = outfmt.Human(stdout, resp.Raw)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func putString(body map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		body[key] = value
	}
}

func putList(body map[string]any, key string, values stringList) {
	if len(values) > 0 {
		body[key] = []string(values)
	}
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func usage(w io.Writer, version string) {
	fmt.Fprintf(w, `exa-cli %s

Usage:
  exa-cli [--json] [--timeout 60s] <command> [flags]

Commands:
  auth [api-key]             Save an Exa API key in ~/.exa-cli/config.json
  auth status                Show credential source
  auth logout                Remove saved credentials
  search "query"             Run web_search_exa
  fetch URL...               Run web_fetch_exa
  advanced-search "query"    Run web_search_advanced_exa
  version                    Print version

Environment:
  EXA_API_KEY                Preferred credential source
  EXA_CLI_CONFIG             Override config path
  EXA_API_BASE               Override API base URL

`, version)
}

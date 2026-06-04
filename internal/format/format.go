package format

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type response struct {
	RequestID          string   `json:"requestId"`
	AutopromptString   string   `json:"autopromptString"`
	ResolvedSearchType string   `json:"resolvedSearchType"`
	Context            string   `json:"context"`
	Results            []result `json:"results"`
}

type result struct {
	ID            string          `json:"id"`
	URL           string          `json:"url"`
	Title         string          `json:"title"`
	Author        string          `json:"author"`
	PublishedDate string          `json:"publishedDate"`
	Text          string          `json:"text"`
	Summary       string          `json:"summary"`
	Highlights    []string        `json:"highlights"`
	Score         *float64        `json:"score"`
	Extras        json.RawMessage `json:"extras"`
	Subpages      []result        `json:"subpages"`
	Error         string          `json:"error"`
	Failed        *bool           `json:"failed"`
	Livecrawl     string          `json:"livecrawl"`
}

func JSON(w io.Writer, raw json.RawMessage) error {
	raw = bytesTrim(raw)
	if len(raw) == 0 {
		_, err := fmt.Fprintln(w, "{}")
		return err
	}
	_, err := w.Write(append(raw, '\n'))
	return err
}

func Human(w io.Writer, raw json.RawMessage) error {
	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Results) == 0 {
		var pretty any
		if json.Unmarshal(raw, &pretty) != nil {
			_, writeErr := fmt.Fprintln(w, strings.TrimSpace(string(raw)))
			return writeErr
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(pretty)
	}

	if parsed.RequestID != "" {
		fmt.Fprintf(w, "Request: %s\n", parsed.RequestID)
	}
	if parsed.AutopromptString != "" {
		fmt.Fprintf(w, "Autoprompt: %s\n", parsed.AutopromptString)
	}
	if parsed.ResolvedSearchType != "" {
		fmt.Fprintf(w, "Search type: %s\n", parsed.ResolvedSearchType)
	}
	if parsed.Context != "" {
		fmt.Fprintf(w, "\nContext:\n%s\n", strings.TrimSpace(parsed.Context))
	}
	if parsed.RequestID != "" || parsed.AutopromptString != "" || parsed.ResolvedSearchType != "" || parsed.Context != "" {
		fmt.Fprintln(w)
	}

	for i, r := range parsed.Results {
		printResult(w, i+1, r, "")
	}
	return nil
}

func printResult(w io.Writer, index int, r result, indent string) {
	title := strings.TrimSpace(r.Title)
	if title == "" {
		title = strings.TrimSpace(r.URL)
	}
	if title == "" {
		title = "(untitled)"
	}
	fmt.Fprintf(w, "%s%d. %s\n", indent, index, title)
	if r.URL != "" {
		fmt.Fprintf(w, "%s   URL: %s\n", indent, r.URL)
	}
	if r.PublishedDate != "" {
		fmt.Fprintf(w, "%s   Published: %s\n", indent, r.PublishedDate)
	}
	if r.Author != "" {
		fmt.Fprintf(w, "%s   Author: %s\n", indent, r.Author)
	}
	if r.Score != nil {
		fmt.Fprintf(w, "%s   Score: %.4f\n", indent, *r.Score)
	}
	if r.Livecrawl != "" {
		fmt.Fprintf(w, "%s   Livecrawl: %s\n", indent, r.Livecrawl)
	}
	if r.Failed != nil && *r.Failed {
		fmt.Fprintf(w, "%s   Failed: true\n", indent)
	}
	if r.Error != "" {
		fmt.Fprintf(w, "%s   Error: %s\n", indent, r.Error)
	}
	if r.Summary != "" {
		fmt.Fprintf(w, "%s   Summary: %s\n", indent, oneLine(r.Summary, 500))
	}
	if r.Text != "" {
		fmt.Fprintf(w, "%s   Text: %s\n", indent, oneLine(r.Text, 700))
	}
	if len(r.Highlights) > 0 {
		fmt.Fprintf(w, "%s   Highlights:\n", indent)
		for _, h := range r.Highlights {
			fmt.Fprintf(w, "%s   - %s\n", indent, oneLine(h, 350))
		}
	}
	for i, subpage := range r.Subpages {
		if i == 0 {
			fmt.Fprintf(w, "%s   Subpages:\n", indent)
		}
		printResult(w, i+1, subpage, indent+"   ")
	}
	fmt.Fprintln(w)
}

func oneLine(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if max > 0 && len(value) > max {
		return value[:max-1] + "..."
	}
	return value
}

func bytesTrim(raw json.RawMessage) json.RawMessage {
	return json.RawMessage(strings.TrimSpace(string(raw)))
}

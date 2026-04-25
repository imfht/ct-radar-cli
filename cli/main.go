package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"
)

const defaultURL = "https://cert.imfht.com/api/search"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Result struct {
	Domain string `json:"domain"`
}

type Response struct {
	Results            []Result `json:"results"`
	Error              string   `json:"error"`
	Message            string   `json:"message"`
	RetryAfterSeconds  int      `json:"retry_after_seconds"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `ct-radar — search Certificate Transparency logs for domains.

Usage:
  ct-radar [flags] <domain>

Anonymous: 1 request/min (no signup needed, perfect for ct-radar X.com | httpx).
Free signup at https://cert.imfht.com gets 100/day.
Pro $29/mo gets 10,000/day + Monitor + CSV export.

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Env:
  CT_RADAR_KEY  Optional API key (anonymous works at 1/min)
  CT_RADAR_URL  Override API URL (default %s)

Examples:
  ct-radar example.com
  ct-radar example.com --limit 200 --exclude-expired
  ct-radar example.com | httpx -silent | nuclei -t exposures/
  CT_RADAR_KEY=ctr_xxx ct-radar example.com --limit 500
`, defaultURL)
	}

	limit := flag.Int("limit", 100, "Results limit (max 50 anonymous / 500 pro)")
	offset := flag.Int("offset", 0, "Offset for pagination")
	apiURL := flag.String("url", envOr("CT_RADAR_URL", defaultURL), "API URL")
	apiKey := flag.String("key", os.Getenv("CT_RADAR_KEY"), "API key (or CT_RADAR_KEY env)")
	excludeExpired := flag.Bool("exclude-expired", false, "Hide expired certificates")
	excludeWildcard := flag.Bool("exclude-wildcard", false, "Hide wildcard (*.) certificates")
	retry := flag.Bool("retry-on-throttle", false, "On 429 anonymous rate limit, sleep retry_after and retry once")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ct-radar %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	query := args[0]
	if _, err := strconv.Atoi(query); err == nil {
		fmt.Fprintln(os.Stderr, "Error: query looks like a number; pass a domain like example.com")
		os.Exit(1)
	}

	resp, err := doRequest(*apiURL, *apiKey, query, *limit, *offset, *excludeExpired, *excludeWildcard)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 429 throttle handling — anonymous 1/min is a common case for piped use
	if resp.StatusCode == 429 && *retry {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var data Response
		_ = json.Unmarshal(body, &data)
		wait := data.RetryAfterSeconds
		if wait == 0 {
			if h := resp.Header.Get("Retry-After"); h != "" {
				if w, err := strconv.Atoi(h); err == nil {
					wait = w
				}
			}
		}
		if wait == 0 {
			wait = 60
		}
		fmt.Fprintf(os.Stderr, "# rate-limited, waiting %ds then retrying...\n", wait)
		time.Sleep(time.Duration(wait+1) * time.Second)
		resp, err = doRequest(*apiURL, *apiKey, query, *limit, *offset, *excludeExpired, *excludeWildcard)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		body, _ := io.ReadAll(resp.Body)
		var data Response
		_ = json.Unmarshal(body, &data)
		msg := data.Message
		if msg == "" {
			msg = data.Error
		}
		if msg == "" {
			msg = "rate limited"
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		if data.RetryAfterSeconds > 0 {
			fmt.Fprintf(os.Stderr, "  retry after %ds, or use -retry-on-throttle, or set CT_RADAR_KEY for higher quota.\n", data.RetryAfterSeconds)
		}
		os.Exit(2)
	}
	if resp.StatusCode == 401 {
		fmt.Fprintln(os.Stderr, "Error: invalid API key (check CT_RADAR_KEY)")
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: API returned %d: %s\n", resp.StatusCode, truncate(string(body), 200))
		os.Exit(1)
	}

	var data Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	// Pipe-friendly: one unique domain per line, sorted
	uniq := make(map[string]bool)
	for _, r := range data.Results {
		if r.Domain != "" {
			uniq[r.Domain] = true
		}
	}
	sorted := make([]string, 0, len(uniq))
	for d := range uniq {
		sorted = append(sorted, d)
	}
	sort.Strings(sorted)
	for _, d := range sorted {
		fmt.Println(d)
	}
}

func doRequest(apiURL, apiKey, query string, limit, offset int, excludeExpired, excludeWildcard bool) (*http.Response, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if excludeExpired {
		q.Set("exclude_expired", "true")
	}
	if excludeWildcard {
		q.Set("exclude_wildcard", "true")
	}
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ct-radar-cli/"+version)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

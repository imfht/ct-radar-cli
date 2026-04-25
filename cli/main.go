package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Result struct {
	Domain string `json:"domain"`
}

type Response struct {
	Results []Result `json:"results"`
	Error   string   `json:"error"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <domain> [limit]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s --version\n", os.Args[0])
		os.Exit(1)
	}

	if os.Args[1] == "--version" || os.Args[1] == "-v" {
		fmt.Printf("ct-radar %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}

	query := os.Args[1]
	limit := "100"
	if len(os.Args) > 2 {
		if _, err := strconv.Atoi(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: limit must be an integer\n")
			os.Exit(1)
		}
		limit = os.Args[2]
	}

	apiKey := os.Getenv("CT_RADAR_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: CT_RADAR_KEY environment variable is required")
		os.Exit(1)
	}

	// 构造请求 URL (指向我们的 SaaS Proxy)
	apiURL := "http://localhost:3000/api/search"
	u, _ := url.Parse(apiURL)
	q := u.Query()
	q.Set("q", query)
	q.Set("limit", limit)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "API returned error: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	var data Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	// 提取并去重域名
	uniqueDomains := make(map[string]bool)
	for _, r := range data.Results {
		if r.Domain != "" {
			uniqueDomains[r.Domain] = true
		}
	}

	// 排序并打印到 STDOUT
	var sorted []string
	for d := range uniqueDomains {
		sorted = append(sorted, d)
	}
	sort.Strings(sorted)

	for _, d := range sorted {
		fmt.Println(d)
	}
}

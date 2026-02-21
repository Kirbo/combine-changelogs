package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ReleasedAt  time.Time `json:"released_at"`
}

type GitLabClient struct {
	baseURL     string
	token       string
	tokenHeader string // "PRIVATE-TOKEN" or "JOB-TOKEN"
	httpClient  *http.Client
}

func NewGitLabClient(gitlabURL, token, tokenHeader string) *GitLabClient {
	return &GitLabClient{
		baseURL:     strings.TrimRight(gitlabURL, "/") + "/api/v4",
		token:       token,
		tokenHeader: tokenHeader,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *GitLabClient) get(endpoint string, params url.Values) (*http.Response, error) {
	reqURL := c.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set(c.tokenHeader, c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

func (c *GitLabClient) FetchAllReleases(projectPath string) ([]Release, error) {
	encodedProject := url.PathEscape(projectPath)
	endpoint := fmt.Sprintf("/projects/%s/releases", encodedProject)

	var allReleases []Release
	page := 1
	perPage := 20

	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("per_page", strconv.Itoa(perPage))

		resp, err := c.get(endpoint, params)
		if err != nil {
			return nil, fmt.Errorf("fetching page %d: %w", page, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitLab API returned %d: %s", resp.StatusCode, string(body))
		}

		var releases []Release
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, fmt.Errorf("parsing releases: %w", err)
		}

		allReleases = append(allReleases, releases...)

		nextPage := resp.Header.Get("X-Next-Page")
		if nextPage == "" {
			break
		}

		next, err := strconv.Atoi(nextPage)
		if err != nil {
			break
		}
		page = next
	}

	return allReleases, nil
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// versionHeading matches a markdown heading whose text starts with a version
// number (e.g. "## 1.2.3 (2024-01-15)", "# v2.0.0 - 2024-03-01", "## [1.0]").
var versionHeading = regexp.MustCompile(`^#{1,6}\s+v?\[?\d[\d.]*`)

// commitRef matches a short or full commit hash wrapped in parentheses,
// e.g. (04cb6bd2) or (04cb6bd2e1f3). Requires 7–40 lowercase hex digits to
// avoid false positives on short numeric strings or CSS colours.
var commitRef = regexp.MustCompile(`\(([0-9a-f]{7,40})\)`)

// stripLeadingVersionHeader removes the first line of a description when it is
// a markdown heading that duplicates the release name/version already written
// by the caller (e.g. "## 1.2.3 (2024-01-15)").
func stripLeadingVersionHeader(description string) string {
	first, rest, _ := strings.Cut(description, "\n")
	if !versionHeading.MatchString(first) {
		return description
	}
	return strings.TrimSpace(rest)
}

// linkifyCommits replaces bare commit-hash references like (04cb6bd2) with
// a markdown link to the commit page on GitLab.
func linkifyCommits(s, commitBaseURL string) string {
	return commitRef.ReplaceAllStringFunc(s, func(match string) string {
		hash := match[1 : len(match)-1] // strip surrounding parens
		return fmt.Sprintf("([%s](%s/%s))", hash, commitBaseURL, hash)
	})
}

func writeChangelog(releases []Release, outputPath, commitBaseURL string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	w := func(format string, args ...any) {
		fmt.Fprintf(f, format, args...)
	}

	w("# Changelog\n\n")
	w("All notable changes to this project will be documented in this file.\n\n")

	if len(releases) == 0 {
		w("No releases found.\n")
		return nil
	}

	for _, r := range releases {
		name := r.Name
		if name == "" {
			name = r.TagName
		}

		date := r.ReleasedAt
		if date.IsZero() {
			date = r.CreatedAt
		}

		w("## %s - (%s)\n\n", name, formatDate(date))

		description := linkifyCommits(stripLeadingVersionHeader(strings.TrimSpace(r.Description)), commitBaseURL)
		if description != "" {
			w("%s\n\n", description)
		} else {
			w("_No description provided._\n\n")
		}
	}

	return nil
}

func resolveEnv(flagVal, envKey, fallback string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

func main() {
	var (
		flagURL     = flag.String("url", "", "GitLab instance URL (default: $CI_SERVER_URL or https://gitlab.com)")
		projectPath = flag.String("project", "", "GitLab project path (e.g. group/project) or numeric ID (default: $CI_PROJECT_PATH)")
		token       = flag.String("token", "", "GitLab private token (or set GITLAB_TOKEN / CI_JOB_TOKEN env var)")
		output      = flag.String("output", "CHANGELOG.md", "Output file path")
	)
	flag.Parse()

	// Resolve project: flag > CI_PROJECT_PATH
	project := resolveEnv(*projectPath, "CI_PROJECT_PATH", "")
	if project == "" {
		fmt.Fprintln(os.Stderr, "Error: -project flag is required (or set CI_PROJECT_PATH)")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Resolve URL: flag > CI_SERVER_URL > default
	gitlabURL := resolveEnv(*flagURL, "CI_SERVER_URL", "https://gitlab.com")

	// Resolve token: -token flag > GITLAB_TOKEN > CI_JOB_TOKEN
	// CI_JOB_TOKEN requires a different auth header ("JOB-TOKEN" vs "PRIVATE-TOKEN").
	resolvedToken := *token
	tokenHeader := "PRIVATE-TOKEN"
	if resolvedToken == "" {
		if v := os.Getenv("GITLAB_TOKEN"); v != "" {
			resolvedToken = v
		} else if v := os.Getenv("CI_JOB_TOKEN"); v != "" {
			resolvedToken = v
			tokenHeader = "JOB-TOKEN"
		}
	}

	client := NewGitLabClient(gitlabURL, resolvedToken, tokenHeader)

	log.Printf("Fetching releases for project: %s", project)
	releases, err := client.FetchAllReleases(project)
	if err != nil {
		log.Fatalf("Error fetching releases: %v", err)
	}

	log.Printf("Found %d release(s)", len(releases))

	commitBaseURL := strings.TrimRight(gitlabURL, "/") + "/" + project + "/-/commit"
	if err := writeChangelog(releases, *output, commitBaseURL); err != nil {
		log.Fatalf("Error writing changelog: %v", err)
	}

	log.Printf("Changelog written to %s", *output)
}

package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// GitLabSource fetches releases from a GitLab project via the REST API.
type GitLabSource struct {
	instanceURL string // e.g. "https://gitlab.com" (no trailing slash)
	baseURL     string
	project     string
	token       string
	tokenHeader string // "PRIVATE-TOKEN" or "JOB-TOKEN"
	httpClient  *http.Client
}

// NewGitLabSource creates a GitLabSource for the given instance URL, project,
// and authentication token. tokenHeader must be "PRIVATE-TOKEN" or "JOB-TOKEN".
func NewGitLabSource(instanceURL, project, token, tokenHeader string) *GitLabSource {
	trimmed := strings.TrimRight(instanceURL, "/")
	return &GitLabSource{
		instanceURL: trimmed,
		baseURL:     trimmed + "/api/v4",
		project:     project,
		token:       token,
		tokenHeader: tokenHeader,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// resolveEnv returns flagVal when non-empty, otherwise the value of the
// environment variable envKey, otherwise fallback.
func resolveEnv(flagVal, envKey, fallback string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

// NewGitLabSourceFromEnv creates a GitLabSource, resolving URL, project, and
// token from environment variables when the corresponding flags are empty:
//
//	URL:     flagURL > $CI_SERVER_URL > "https://gitlab.com"
//	Project: flagProject > $CI_PROJECT_PATH
//	Token:   flagToken > $GITLAB_TOKEN (PRIVATE-TOKEN) > $CI_JOB_TOKEN (JOB-TOKEN)
func NewGitLabSourceFromEnv(flagURL, flagProject, flagToken string) *GitLabSource {
	instanceURL := resolveEnv(flagURL, "CI_SERVER_URL", "https://gitlab.com")
	project := resolveEnv(flagProject, "CI_PROJECT_PATH", "")

	token := flagToken
	tokenHeader := "PRIVATE-TOKEN"
	if token == "" {
		if v := os.Getenv("GITLAB_TOKEN"); v != "" {
			token = v
		} else if v := os.Getenv("CI_JOB_TOKEN"); v != "" {
			token = v
			tokenHeader = "JOB-TOKEN"
		}
	}

	return NewGitLabSource(instanceURL, project, token, tokenHeader)
}

// Project returns the resolved GitLab project path (e.g. "group/repo").
func (s *GitLabSource) Project() string { return s.project }

// CommitBaseURL returns the base URL for commit links, or "" when no project
// is set (e.g. local-only mode) so callers can skip linkification.
func (s *GitLabSource) CommitBaseURL() string {
	if s.project == "" {
		return ""
	}
	return s.instanceURL + "/" + s.project + "/-/commit"
}

// FetchReleases returns all releases for the project, fetching every page.
func (s *GitLabSource) FetchReleases() ([]Release, error) {
	encodedProject := url.PathEscape(s.project)
	endpoint := fmt.Sprintf("/projects/%s/releases", encodedProject)

	var allReleases []Release
	page := 1
	perPage := 20

	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("per_page", strconv.Itoa(perPage))

		resp, err := s.get(endpoint, params)
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

func (s *GitLabSource) get(endpoint string, params url.Values) (*http.Response, error) {
	reqURL := s.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if s.token != "" {
		req.Header.Set(s.tokenHeader, s.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

package sources

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	privHeader     = "PRIVATE-TOKEN"
	jobHeader      = "JOB-TOKEN"
	defaultBaseURL = "https://gitlab.com/api/v4"
	testProject    = "foo/bar"
	testToken      = "glpat-secret"
	tagV200        = "v2.0.0"
	tagV100        = "v1.0.0"
)

// releaseNames is a local helper so this package's tests are self-contained.
func releaseNames(releases []Release) []string {
	out := make([]string, len(releases))
	for i, r := range releases {
		out[i] = r.Name
	}
	return out
}

// assertSourceFields checks the resolved fields of a GitLabSource.
func assertSourceFields(t *testing.T, src *GitLabSource, wantBase, wantProject, wantToken, wantHeader string) {
	t.Helper()
	if src.baseURL != wantBase {
		t.Errorf("baseURL: got %q; want %q", src.baseURL, wantBase)
	}
	if src.project != wantProject {
		t.Errorf("project: got %q; want %q", src.project, wantProject)
	}
	if src.token != wantToken {
		t.Errorf("token: got %q; want %q", src.token, wantToken)
	}
	if src.tokenHeader != wantHeader {
		t.Errorf("tokenHeader: got %q; want %q", src.tokenHeader, wantHeader)
	}
}

// ── NewGitLabSourceFromEnv ────────────────────────────────────────────────────

func TestNewGitLabSourceFromEnv(t *testing.T) {
	// No t.Parallel() — subtests use t.Setenv which requires a serial parent.
	cases := []struct {
		name        string
		flagURL     string
		flagProject string
		flagToken   string
		envVars     map[string]string
		wantBase    string
		wantProject string
		wantToken   string
		wantHeader  string
	}{
		{
			name:        "flag values take priority over env",
			flagURL:     "https://mygitlab.com",
			flagProject: "a/b",
			flagToken:   "flagtok",
			envVars:     map[string]string{"CI_SERVER_URL": "https://other.com", "CI_PROJECT_PATH": "c/d", "GITLAB_TOKEN": "envtok"},
			wantBase:    "https://mygitlab.com/api/v4",
			wantProject: "a/b",
			wantToken:   "flagtok",
			wantHeader:  privHeader,
		},
		{
			name:        "falls back to CI env vars with CI_JOB_TOKEN",
			envVars:     map[string]string{"CI_SERVER_URL": "https://ci.example.com", "CI_PROJECT_PATH": "ci/proj", "CI_JOB_TOKEN": "jobtok"},
			wantBase:    "https://ci.example.com/api/v4",
			wantProject: "ci/proj",
			wantToken:   "jobtok",
			wantHeader:  jobHeader,
		},
		{
			name:        "GITLAB_TOKEN preferred over CI_JOB_TOKEN",
			envVars:     map[string]string{"GITLAB_TOKEN": "pattok", "CI_JOB_TOKEN": "jobtok"},
			wantBase:    defaultBaseURL,
			wantProject: "",
			wantToken:   "pattok",
			wantHeader:  privHeader,
		},
		{
			name:       "defaults to gitlab.com with no env or flags",
			wantBase:   defaultBaseURL,
			wantToken:  "",
			wantHeader: privHeader,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for k, v := range c.envVars {
				t.Setenv(k, v)
			}
			src := NewGitLabSourceFromEnv(c.flagURL, c.flagProject, c.flagToken)
			assertSourceFields(t, src, c.wantBase, c.wantProject, c.wantToken, c.wantHeader)
		})
	}
}

// ── NewGitLabSource ───────────────────────────────────────────────────────────

func TestNewGitLabSource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url      string
		wantBase string
	}{
		{"https://gitlab.com", defaultBaseURL},
		{"https://gitlab.com/", defaultBaseURL},
		{"https://gitlab.example.com", "https://gitlab.example.com/api/v4"},
	}
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			t.Parallel()
			src := NewGitLabSource(c.url, "proj", "tok", privHeader)
			if src.baseURL != c.wantBase {
				t.Errorf("baseURL: got %q; want %q", src.baseURL, c.wantBase)
			}
		})
	}
}

// ── GitLabSource auth headers ─────────────────────────────────────────────────

func TestGitLabSourceAuthHeaders(t *testing.T) {
	t.Parallel()

	t.Run("PRIVATE-TOKEN header sent", func(t *testing.T) {
		t.Parallel()
		var got string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.Header.Get(privHeader)
			w.Write([]byte("[]")) //nolint:errcheck
		}))
		defer srv.Close()

		NewGitLabSource(srv.URL, testProject, testToken, privHeader).FetchReleases() //nolint:errcheck
		if got != testToken {
			t.Errorf(privHeader+": got %q; want %q", got, testToken)
		}
	})

	t.Run("JOB-TOKEN header sent", func(t *testing.T) {
		t.Parallel()
		var got string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.Header.Get(jobHeader)
			w.Write([]byte("[]")) //nolint:errcheck
		}))
		defer srv.Close()

		NewGitLabSource(srv.URL, testProject, "cijobtoken", jobHeader).FetchReleases() //nolint:errcheck
		if got != "cijobtoken" {
			t.Errorf(jobHeader+": got %q; want %q", got, "cijobtoken")
		}
	})

	t.Run("no token means no auth header", func(t *testing.T) {
		t.Parallel()
		var privateHeader, gotJobHeader string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			privateHeader = r.Header.Get(privHeader)
			gotJobHeader = r.Header.Get(jobHeader)
			w.Write([]byte("[]")) //nolint:errcheck
		}))
		defer srv.Close()

		NewGitLabSource(srv.URL, testProject, "", privHeader).FetchReleases() //nolint:errcheck
		if privateHeader != "" || gotJobHeader != "" {
			t.Errorf("expected no auth header; got PRIVATE-TOKEN=%q JOB-TOKEN=%q", privateHeader, gotJobHeader)
		}
	})
}

// ── GitLabSource.FetchReleases ────────────────────────────────────────────────

func TestGitLabSourceFetchReleasesSinglePage(t *testing.T) {
	t.Parallel()
	want := []Release{
		{TagName: tagV200, Name: "Release 2.0.0"},
		{TagName: tagV100, Name: "Release 1.0.0"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	got, err := NewGitLabSource(srv.URL, testProject, "", privHeader).FetchReleases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Errorf("expected %d releases, got %d", len(want), len(got))
	}
	if got[0].TagName != tagV200 {
		t.Errorf("first tag: got %q; want %s", got[0].TagName, tagV200)
	}
}

func TestGitLabSourceFetchReleasesPagination(t *testing.T) {
	t.Parallel()
	page1 := []Release{{TagName: tagV200}}
	page2 := []Release{{TagName: tagV100}}
	paginationHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "2" {
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode(page1) //nolint:errcheck
		} else {
			json.NewEncoder(w).Encode(page2) //nolint:errcheck
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(paginationHandler))
	defer srv.Close()

	got, err := NewGitLabSource(srv.URL, testProject, "", privHeader).FetchReleases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 releases across pages, got %d", len(got))
	}
	if got[0].TagName != tagV200 || got[1].TagName != tagV100 {
		t.Errorf("unexpected tags: %v", releaseNames(got))
	}
}

func TestGitLabSourceFetchReleasesErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "non-200 response returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			},
		},
		{
			name: "invalid JSON returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("not json at all")) //nolint:errcheck
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(c.handler)
			defer srv.Close()
			_, err := NewGitLabSource(srv.URL, testProject, "", privHeader).FetchReleases()
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestGitLabSourceFetchReleasesRequestFormat(t *testing.T) {
	t.Parallel()

	t.Run("project path is URL-encoded", func(t *testing.T) {
		t.Parallel()
		var gotURI string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// r.RequestURI is the raw uninterpreted request-target as sent by the
			// client, so it preserves percent-encoding unlike r.URL.Path.
			gotURI = r.RequestURI
			w.Write([]byte("[]")) //nolint:errcheck
		}))
		defer srv.Close()

		NewGitLabSource(srv.URL, "my group/my repo", "", privHeader).FetchReleases() //nolint:errcheck
		if !strings.Contains(gotURI, "my%20group") {
			t.Errorf("expected URL-encoded path in RequestURI, got %q", gotURI)
		}
	})

	t.Run("Accept header is application/json", func(t *testing.T) {
		t.Parallel()
		var gotAccept string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAccept = r.Header.Get("Accept")
			w.Write([]byte("[]")) //nolint:errcheck
		}))
		defer srv.Close()

		NewGitLabSource(srv.URL, testProject, "", privHeader).FetchReleases() //nolint:errcheck
		if gotAccept != "application/json" {
			t.Errorf("Accept: got %q; want application/json", gotAccept)
		}
	})
}

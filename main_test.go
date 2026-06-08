package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"combine-changelogs/sources"
)

// ── helpers ──────────────────────────────────────────────────────────────────

const (
	fmtGotWant   = "got %q; want %q"
	fmtExpected1 = "expected 1 release, got %d"
	nameField    = "name: "
	v100         = "v1.0.0"
	date20240115 = "2024-01-15"
)

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertRelease(t *testing.T, r sources.Release, wantName, wantTag, wantDate, wantDescSub string) {
	t.Helper()
	if r.Name != wantName {
		t.Errorf(nameField+fmtGotWant, r.Name, wantName)
	}
	if r.TagName != wantTag {
		t.Errorf("tag_name: "+fmtGotWant, r.TagName, wantTag)
	}
	if got := formatDate(r.ReleasedAt); got != wantDate {
		t.Errorf("ReleasedAt: "+fmtGotWant, got, wantDate)
	}
	if !strings.Contains(r.Description, wantDescSub) {
		t.Errorf("description missing %q, got: %q", wantDescSub, r.Description)
	}
}

// mustParseChangelog parses content via a temp file, failing on error.
func mustParseChangelog(t *testing.T, content string) []sources.Release {
	t.Helper()
	releases, err := parseChangelogFile(writeTempChangelog(t, content))
	assertNoError(t, err)
	return releases
}

// mustWriteChangelog writes a changelog to a temp file and returns its content.
func mustWriteChangelog(t *testing.T, releases []sources.Release, base string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.md")
	assertNoError(t, writeChangelog(releases, path, base))
	return readFile(t, path)
}

func writeTempChangelog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTempChangelog: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile: %v", err)
	}
	return string(data)
}

func releaseNames(releases []sources.Release) []string {
	out := make([]string, len(releases))
	for i, r := range releases {
		out[i] = r.Name
	}
	return out
}

// ── stringSlice ───────────────────────────────────────────────────────────────

func TestStringSlice(t *testing.T) {
	t.Parallel()
	var s stringSlice

	if got := s.String(); got != "" {
		t.Errorf("empty String(): got %q; want %q", got, "")
	}

	if err := s.Set("alpha"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Set("beta"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if got, want := s.String(), "alpha, beta"; got != want {
		t.Errorf("String(): "+fmtGotWant, got, want)
	}
	if len(s) != 2 || s[0] != "alpha" || s[1] != "beta" {
		t.Errorf("unexpected slice contents: %v", []string(s))
	}
}

// ── formatDate ───────────────────────────────────────────────────────────────

func TestFormatDate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   time.Time
		want string
	}{
		{time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC), date20240115},
		{time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC), "2023-12-31"},
		{time.Date(2000, 6, 1, 0, 0, 0, 0, time.UTC), "2000-06-01"},
	}
	for _, c := range cases {
		if got := formatDate(c.in); got != c.want {
			t.Errorf("formatDate(%v) = %q; want %q", c.in, got, c.want)
		}
	}
}

// ── stripLeadingVersionHeader ─────────────────────────────────────────────────

func TestStripLeadingVersionHeader(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "version heading stripped",
			in:   "## 1.2.3 (2024-01-15)\n\nSome content",
			want: "Some content",
		},
		{
			name: "v-prefix heading stripped",
			in:   "## v2.0.0\n\nContent here",
			want: "Content here",
		},
		{
			name: "bracketed version heading stripped",
			in:   "## [1.0.0] - 2024-01-15\n\nBody",
			want: "Body",
		},
		{
			name: "non-version heading preserved",
			in:   "## Changes\n\nContent",
			want: "## Changes\n\nContent",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "only a version heading, no body",
			in:   "## 1.0.0",
			want: "",
		},
		{
			name: "description without any heading preserved",
			in:   "* fix: something\n* feat: something else",
			want: "* fix: something\n* feat: something else",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := stripLeadingVersionHeader(c.in); got != c.want {
				t.Errorf(fmtGotWant, got, c.want)
			}
		})
	}
}

// ── linkifyCommits ────────────────────────────────────────────────────────────

func TestLinkifyCommits(t *testing.T) {
	t.Parallel()
	base := "https://gitlab.com/foo/bar/-/commit"

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "7-char hash linkified",
			in:   "fix (04cb6bd)",
			want: "fix ([04cb6bd](https://gitlab.com/foo/bar/-/commit/04cb6bd))",
		},
		{
			name: "8-char hash linkified",
			in:   "add feature (04cb6bd2)",
			want: "add feature ([04cb6bd2](https://gitlab.com/foo/bar/-/commit/04cb6bd2))",
		},
		{
			name: "40-char full hash linkified",
			in:   "commit (da39a3ee5e6b4b0d3255bfef95601890afd80709)",
			want: "commit ([da39a3ee5e6b4b0d3255bfef95601890afd80709](https://gitlab.com/foo/bar/-/commit/da39a3ee5e6b4b0d3255bfef95601890afd80709))",
		},
		{
			name: "6-char hash not linkified (below minimum)",
			in:   "old (abcdef)",
			want: "old (abcdef)",
		},
		{
			name: "41-char hash not linkified (above maximum)",
			in:   "long (da39a3ee5e6b4b0d3255bfef95601890afd807090)",
			want: "long (da39a3ee5e6b4b0d3255bfef95601890afd807090)",
		},
		{
			name: "uppercase hex not linkified",
			in:   "upper (ABCDEF1234567)",
			want: "upper (ABCDEF1234567)",
		},
		{
			name: "multiple hashes in one line",
			in:   "fix (04cb6bd) revert (deadbeef)",
			want: "fix ([04cb6bd](https://gitlab.com/foo/bar/-/commit/04cb6bd)) revert ([deadbeef](https://gitlab.com/foo/bar/-/commit/deadbeef))",
		},
		{
			name: "no hash — unchanged",
			in:   "just plain text",
			want: "just plain text",
		},
		{
			name: "already-linked hash not double-linked",
			in:   "([04cb6bd](https://gitlab.com/foo/bar/-/commit/04cb6bd))",
			want: "([04cb6bd](https://gitlab.com/foo/bar/-/commit/04cb6bd))",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := linkifyCommits(c.in, base); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

// ── parseVersionHeading ───────────────────────────────────────────────────────

func TestParseVersionHeading(t *testing.T) {
	t.Parallel()
	today := formatDate(time.Now())

	cases := []struct {
		name     string
		line     string
		wantName string
		wantDate string // YYYY-MM-DD
	}{
		{"go-semantic-release format", "## 1.2.3 (2024-01-15)", "1.2.3", date20240115},
		{"v-prefix with date", "## v2.0.0 (2023-06-01)", "v2.0.0", "2023-06-01"},
		{"single hash heading", "# 1.0.0 (2022-12-31)", "1.0.0", "2022-12-31"},
		{"triple hash heading", "### 3.0.0 (2024-06-15)", "3.0.0", "2024-06-15"},
		{"no date falls back to today", "## 1.0.0", "1.0.0", today},
		// headerDate requires "(YYYY-MM-DD)"; "- YYYY-MM-DD" is not matched.
		{"keep-a-changelog dash-date not parsed", "## [1.2.3] - 2024-01-15", "[1.2.3] - 2024-01-15", today},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			name, date := parseVersionHeading(c.line)
			if name != c.wantName {
				t.Errorf(nameField+fmtGotWant, name, c.wantName)
			}
			if got := formatDate(date); got != c.wantDate {
				t.Errorf("date: "+fmtGotWant, got, c.wantDate)
			}
		})
	}
}

// ── parseChangelogFile ────────────────────────────────────────────────────────

func TestParseChangelogFile(t *testing.T) {
	t.Parallel()

	// Table: given content, expect N releases.
	countCases := []struct {
		name    string
		content string
		wantN   int
	}{
		{"preamble lines skipped", "# Changelog\n\nAll notable changes.\n\n## 1.0.0 (2024-01-01)\n\n* initial\n", 1},
		{"empty file returns no releases", "", 0},
		{"file with only preamble returns no releases", "# Changelog\n\nNothing yet.\n", 0},
		{"go-semantic-release subsections counted as one", "## 1.2.3 (2024-01-15)\n\n### Features\n\n* feat: thing\n\n### Bug Fixes\n\n* fix: bug\n", 1},
	}
	for _, c := range countCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			releases := mustParseChangelog(t, c.content)
			if len(releases) != c.wantN {
				t.Fatalf("expected %d release(s), got %d", c.wantN, len(releases))
			}
		})
	}

	t.Run("single entry fields", func(t *testing.T) {
		t.Parallel()
		releases := mustParseChangelog(t, "## 1.0.0 (2024-01-15)\n\n* feat: initial release\n")
		if len(releases) != 1 {
			t.Fatalf(fmtExpected1, len(releases))
		}
		assertRelease(t, releases[0], "1.0.0", "1.0.0", date20240115, "initial release")
	})

	t.Run("multiple entries preserve order", func(t *testing.T) {
		t.Parallel()
		content := "## 1.2.0 (2024-03-01)\n\n* feat: new\n\n## 1.1.0 (2024-02-01)\n\n* fix: old\n"
		releases := mustParseChangelog(t, content)
		if len(releases) != 2 {
			t.Fatalf("expected 2 releases, got %d", len(releases))
		}
		if releases[0].Name != "1.2.0" || releases[1].Name != "1.1.0" {
			t.Errorf("wrong order: %v", releaseNames(releases))
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		t.Parallel()
		_, err := parseChangelogFile("/nonexistent/path/CHANGELOG.md")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

// ── sortReleases ──────────────────────────────────────────────────────────────

func TestSortReleases(t *testing.T) {
	t.Parallel()

	d := func(year int) time.Time {
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	t.Run("sorts newest-first by ReleasedAt", func(t *testing.T) {
		t.Parallel()
		releases := []sources.Release{
			{Name: "old", ReleasedAt: d(2022)},
			{Name: "new", ReleasedAt: d(2024)},
			{Name: "mid", ReleasedAt: d(2023)},
		}
		sortReleases(releases)
		want := []string{"new", "mid", "old"}
		if got := releaseNames(releases); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("order: got %v; want %v", got, want)
		}
	})

	t.Run("falls back to CreatedAt when ReleasedAt is zero", func(t *testing.T) {
		t.Parallel()
		releases := []sources.Release{
			{Name: "a", CreatedAt: d(2022)},
			{Name: "b", CreatedAt: d(2024)},
		}
		sortReleases(releases)
		if releases[0].Name != "b" {
			t.Errorf("expected b first, got %q", releases[0].Name)
		}
	})

	t.Run("ReleasedAt takes priority over CreatedAt", func(t *testing.T) {
		t.Parallel()
		releases := []sources.Release{
			{Name: "late-release", CreatedAt: d(2020), ReleasedAt: d(2024)},
			{Name: "early-release", CreatedAt: d(2024), ReleasedAt: d(2020)},
		}
		sortReleases(releases)
		if releases[0].Name != "late-release" {
			t.Errorf("expected late-release first (ReleasedAt priority), got %q", releases[0].Name)
		}
	})

	t.Run("single release unchanged", func(t *testing.T) {
		t.Parallel()
		releases := []sources.Release{{Name: "only", ReleasedAt: d(2024)}}
		sortReleases(releases)
		if releases[0].Name != "only" {
			t.Errorf("unexpected name: %q", releases[0].Name)
		}
	})

	t.Run("empty slice does not panic", func(t *testing.T) {
		t.Parallel()
		sortReleases(nil)
		sortReleases([]sources.Release{})
	})
}

// ── writeChangelog ────────────────────────────────────────────────────────────

func TestWriteChangelog(t *testing.T) {
	t.Parallel()
	const base = "https://gitlab.com/foo/bar/-/commit"

	// Table: given releases, expect output to contain wantSubstr.
	containsCases := []struct {
		name       string
		releases   []sources.Release
		wantSubstr string
	}{
		{
			"empty releases writes placeholder",
			nil,
			"No releases found",
		},
		{
			"release heading uses ReleasedAt",
			[]sources.Release{{Name: v100, Description: "* initial", ReleasedAt: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)}},
			"## v1.0.0 - (2024-03-15)",
		},
		{
			"falls back to CreatedAt when ReleasedAt is zero",
			[]sources.Release{{Name: v100, CreatedAt: time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)}},
			"2023-06-15",
		},
		{
			"falls back to TagName when Name is empty",
			[]sources.Release{{TagName: "v1.2.3", Description: "* fix", ReleasedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
			"v1.2.3",
		},
		{
			"empty description writes placeholder",
			[]sources.Release{{Name: v100, ReleasedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
			"_No description provided._",
		},
		{
			"commit hashes in description are linkified",
			[]sources.Release{{Name: v100, Description: "* fix: something (04cb6bd2)", ReleasedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
			"[04cb6bd2]",
		},
	}
	for _, c := range containsCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(mustWriteChangelog(t, c.releases, base), c.wantSubstr) {
				t.Errorf("output missing %q", c.wantSubstr)
			}
		})
	}

	t.Run("changelog structure", func(t *testing.T) {
		t.Parallel()
		// A release whose description starts with a duplicate version heading.
		releases := []sources.Release{{Name: v100, Description: "## v1.0.0\n\n* initial", ReleasedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}
		content := mustWriteChangelog(t, releases, base)
		if !strings.HasPrefix(content, "# Changelog\n") {
			t.Errorf("expected '# Changelog' header, got: %q", content[:min(len(content), 60)])
		}
		if strings.Count(content, v100) != 1 {
			t.Errorf("expected %q once (stripped duplicate); got %d times", v100, strings.Count(content, v100))
		}
	})

	t.Run("invalid output path returns error", func(t *testing.T) {
		t.Parallel()
		if err := writeChangelog(nil, "/nonexistent/dir/out.md", base); err == nil {
			t.Error("expected an error for invalid path, got nil")
		}
	})
}

// ── loadIncludes ─────────────────────────────────────────────────────────────

func TestLoadIncludes(t *testing.T) {
	t.Parallel()

	t.Run("local file loaded", func(t *testing.T) {
		t.Parallel()
		path := writeTempChangelog(t, "## 1.0.0 (2024-01-15)\n\n* feat: initial release\n")
		releases, err := loadIncludes([]string{path})
		assertNoError(t, err)
		if len(releases) != 1 {
			t.Fatalf(fmtExpected1, len(releases))
		}
		if releases[0].Name != "1.0.0" {
			t.Errorf(nameField+fmtGotWant, releases[0].Name, "1.0.0")
		}
	})

	t.Run("http URL fetched and parsed", func(t *testing.T) {
		t.Parallel()
		content := "## 2.0.0 (2024-06-01)\n\n* feat: from URL\n"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(content)) //nolint:errcheck
		}))
		defer srv.Close()

		releases, err := loadIncludes([]string{srv.URL})
		assertNoError(t, err)
		if len(releases) != 1 {
			t.Fatalf(fmtExpected1, len(releases))
		}
		if releases[0].Name != "2.0.0" {
			t.Errorf(nameField+fmtGotWant, releases[0].Name, "2.0.0")
		}
	})

	t.Run("non-200 URL response returns error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer srv.Close()

		_, err := loadIncludes([]string{srv.URL})
		if err == nil {
			t.Error("expected error for non-200 response, got nil")
		}
	})

	t.Run("nonexistent local file returns error", func(t *testing.T) {
		t.Parallel()
		_, err := loadIncludes([]string{"/nonexistent/path/CHANGELOG.md"})
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})

	t.Run("multiple sources merged", func(t *testing.T) {
		t.Parallel()
		file1 := writeTempChangelog(t, "## 1.0.0 (2024-01-01)\n\n* first\n")
		file2 := writeTempChangelog(t, "## 2.0.0 (2024-06-01)\n\n* second\n")
		releases, err := loadIncludes([]string{file1, file2})
		assertNoError(t, err)
		if len(releases) != 2 {
			t.Fatalf("expected 2 releases, got %d", len(releases))
		}
	})
}

// loadUnreleased: -unreleased sections are dated "now" so they always sort newest.
func TestLoadUnreleased(t *testing.T) {
	// Pin "now". Not parallel — mutates the nowFunc package var.
	fixed := time.Date(2026, 6, 8, 15, 21, 0, 0, time.UTC)
	orig := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = orig }()

	t.Run("stamps now, ignoring the heading date", func(t *testing.T) {
		path := writeTempChangelog(t, "## 2026.6.12 (2026-06-08)\n\n#### Feature\n\n* something (abcd1234)\n")
		releases, err := loadUnreleased([]string{path})
		assertNoError(t, err)
		if len(releases) != 1 {
			t.Fatalf(fmtExpected1, len(releases))
		}
		if !releases[0].ReleasedAt.Equal(fixed) {
			t.Errorf("ReleasedAt: got %v; want %v (now), not the heading's midnight date", releases[0].ReleasedAt, fixed)
		}
		if releases[0].Name != "2026.6.12" {
			t.Errorf(nameField+fmtGotWant, releases[0].Name, "2026.6.12")
		}
	})

	t.Run("sorts above a same-day published release", func(t *testing.T) {
		path := writeTempChangelog(t, "## 2026.6.12 (2026-06-08)\n\n* unreleased\n")
		unrel, err := loadUnreleased([]string{path})
		assertNoError(t, err)
		// A release published earlier the SAME day (real timestamp) — would beat a midnight heading.
		published := sources.Release{Name: "v2026.6.11", ReleasedAt: time.Date(2026, 6, 8, 13, 45, 2, 0, time.UTC)}
		all := append([]sources.Release{published}, unrel...)
		sortReleases(all)
		if all[0].Name != "2026.6.12" {
			t.Errorf("expected unreleased 2026.6.12 on top; got %q", all[0].Name)
		}
	})
}

// ── compileIgnoreRegexes ──────────────────────────────────────────────────────

func TestCompileIgnoreRegexes(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns empty slice, no error", func(t *testing.T) {
		t.Parallel()
		got, err := compileIgnoreRegexes(nil)
		assertNoError(t, err)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(got))
		}
	})

	t.Run("single valid pattern compiles", func(t *testing.T) {
		t.Parallel()
		got, err := compileIgnoreRegexes([]string{`-rc\.\d+`})
		assertNoError(t, err)
		if len(got) != 1 {
			t.Fatalf("expected 1 compiled regex, got %d", len(got))
		}
		if !got[0].MatchString("v1.2.3-rc.1") {
			t.Errorf(`compiled regex did not match "v1.2.3-rc.1"`)
		}
	})

	t.Run("multiple valid patterns compile in order", func(t *testing.T) {
		t.Parallel()
		got, err := compileIgnoreRegexes([]string{`-rc\.\d+`, `-nightly`})
		assertNoError(t, err)
		if len(got) != 2 {
			t.Fatalf("expected 2 compiled regexes, got %d", len(got))
		}
		if !got[0].MatchString("v1.0.0-rc.1") || got[0].MatchString("v1.0.0-nightly") {
			t.Errorf("first regex behaves unexpectedly: %s", got[0].String())
		}
		if !got[1].MatchString("v1.0.0-nightly") || got[1].MatchString("v1.0.0-rc.1") {
			t.Errorf("second regex behaves unexpectedly: %s", got[1].String())
		}
	})

	t.Run("invalid pattern returns error echoing the bad input", func(t *testing.T) {
		t.Parallel()
		_, err := compileIgnoreRegexes([]string{`[`})
		if err == nil {
			t.Fatal("expected error for invalid regex, got nil")
		}
		if !strings.Contains(err.Error(), `-ignore-regex`) {
			t.Errorf("error should mention flag name: %v", err)
		}
		if !strings.Contains(err.Error(), `[`) {
			t.Errorf("error should echo the bad pattern: %v", err)
		}
	})

	t.Run("first invalid pattern short-circuits", func(t *testing.T) {
		t.Parallel()
		_, err := compileIgnoreRegexes([]string{`[`, `also-invalid(`})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// ── filterIgnoredReleases ────────────────────────────────────────────────────

func TestFilterIgnoredReleases(t *testing.T) {
	t.Parallel()

	// mustCompile is a local helper to keep the test tables readable.
	mustCompile := func(t *testing.T, patterns ...string) []*regexp.Regexp {
		t.Helper()
		compiled, err := compileIgnoreRegexes(patterns)
		assertNoError(t, err)
		return compiled
	}

	mk := func(name, tag string) sources.Release {
		return sources.Release{Name: name, TagName: tag}
	}

	t.Run("no patterns returns input unchanged", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{mk("v1.0.0", "v1.0.0"), mk("v1.1.0", "v1.1.0")}
		got := filterIgnoredReleases(in, nil)
		if diff := releaseNames(got); len(diff) != 2 || diff[0] != "v1.0.0" || diff[1] != "v1.1.0" {
			t.Errorf("unexpected output: %v", diff)
		}
	})

	t.Run("pattern matches Name drops release", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{mk("v1.0.0-rc.1", "v1.0.0-rc.1"), mk("v1.0.0", "v1.0.0")}
		got := filterIgnoredReleases(in, mustCompile(t, `-rc\.\d+`))
		names := releaseNames(got)
		if len(names) != 1 || names[0] != "v1.0.0" {
			t.Errorf("expected only v1.0.0 to survive, got %v", names)
		}
	})

	t.Run("pattern matches TagName when Name empty drops release", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{mk("", "v2.0.0-rc.7"), mk("", "v2.0.0")}
		got := filterIgnoredReleases(in, mustCompile(t, `-rc\.\d+`))
		if len(got) != 1 || got[0].TagName != "v2.0.0" {
			t.Errorf("expected only v2.0.0 to survive, got %+v", got)
		}
	})

	t.Run("no match keeps all releases", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{mk("v1.0.0", "v1.0.0"), mk("v1.1.0", "v1.1.0")}
		got := filterIgnoredReleases(in, mustCompile(t, `-rc\.\d+`))
		if len(got) != 2 {
			t.Errorf("expected 2 survivors, got %d", len(got))
		}
	})

	t.Run("multiple patterns union — any match drops", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{
			mk("v1.0.0", "v1.0.0"),
			mk("v1.0.0-rc.1", "v1.0.0-rc.1"),
			mk("v1.0.0-nightly", "v1.0.0-nightly"),
			mk("v1.1.0", "v1.1.0"),
		}
		got := filterIgnoredReleases(in, mustCompile(t, `-rc\.\d+`, `-nightly`))
		names := releaseNames(got)
		want := []string{"v1.0.0", "v1.1.0"}
		if len(names) != len(want) {
			t.Fatalf("expected %d survivors, got %d: %v", len(want), len(names), names)
		}
		for i, n := range names {
			if n != want[i] {
				t.Errorf("index %d: got %q, want %q", i, n, want[i])
			}
		}
	})

	t.Run("preserves order of survivors", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{
			mk("alpha", "alpha"),
			mk("skip-me", "skip-me"),
			mk("beta", "beta"),
			mk("skip-me-too", "skip-me-too"),
			mk("gamma", "gamma"),
		}
		got := filterIgnoredReleases(in, mustCompile(t, `^skip`))
		names := releaseNames(got)
		want := []string{"alpha", "beta", "gamma"}
		if len(names) != len(want) {
			t.Fatalf("expected %d survivors, got %d: %v", len(want), len(names), names)
		}
		for i, n := range names {
			if n != want[i] {
				t.Errorf("index %d: got %q, want %q", i, n, want[i])
			}
		}
	})

	t.Run("partial match — does not require anchors", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{mk("v1.2.3-rc.1", "v1.2.3-rc.1"), mk("v1.2.3", "v1.2.3")}
		got := filterIgnoredReleases(in, mustCompile(t, `-rc\.\d+`))
		if len(got) != 1 || got[0].Name != "v1.2.3" {
			t.Errorf("expected v1.2.3 to survive, got %+v", got)
		}
	})

	t.Run("does not mutate input slice", func(t *testing.T) {
		t.Parallel()
		in := []sources.Release{mk("keep", "keep"), mk("drop-me", "drop-me")}
		_ = filterIgnoredReleases(in, mustCompile(t, `^drop`))
		if len(in) != 2 || in[0].Name != "keep" || in[1].Name != "drop-me" {
			t.Errorf("input slice was mutated: %+v", in)
		}
	})
}

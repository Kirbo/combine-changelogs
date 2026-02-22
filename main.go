package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"combine-changelogs/sources"
)

// stringSlice is a repeatable flag value (flag.Var). Each -include flag
// appends one path to the slice.
type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// versionHeading matches a markdown heading whose text starts with a version
// number (e.g. "## 1.2.3 (2024-01-15)", "# v2.0.0 - 2024-03-01", "## [1.0]").
var versionHeading = regexp.MustCompile(`^#{1,6}\s+v?\[?\d[\d.]*`)

// headerDate extracts a YYYY-MM-DD date written in parentheses inside a
// version heading, e.g. "## 1.2.3 (2024-01-15)".
var headerDate = regexp.MustCompile(`\((\d{4}-\d{2}-\d{2})\)`)

// parseVersionHeading extracts the release name and date from a markdown
// version heading line like "## 1.2.3 (2024-01-15)". Falls back to time.Now()
// when no parseable date is present.
func parseVersionHeading(line string) (name string, date time.Time) {
	rawName := strings.TrimSpace(strings.TrimLeft(line, "#"))
	if m := headerDate.FindStringSubmatchIndex(rawName); m != nil {
		if t, err := time.Parse("2006-01-02", rawName[m[2]:m[3]]); err == nil {
			date = t
		}
		rawName = strings.TrimRight(rawName[:m[0]], " -([")
	}
	if date.IsZero() {
		date = time.Now()
	}
	return strings.TrimSpace(rawName), date
}

// parseChangelogContent parses a changelog string and returns one Release per
// version section found. Sections are delimited by lines that match
// versionHeading. Non-section preamble lines (like "# Changelog") are skipped.
func parseChangelogContent(content string) []sources.Release {
	var releases []sources.Release
	var curName string
	var curDate time.Time
	var descLines []string

	flush := func() {
		if curName == "" {
			return
		}
		releases = append(releases, sources.Release{
			TagName:     curName,
			Name:        curName,
			Description: strings.TrimSpace(strings.Join(descLines, "\n")),
			ReleasedAt:  curDate,
		})
	}

	for line := range strings.SplitSeq(content, "\n") {
		if !versionHeading.MatchString(line) {
			if curName != "" {
				descLines = append(descLines, line)
			}
			continue
		}
		flush()
		descLines = nil
		curName, curDate = parseVersionHeading(line)
	}

	flush()
	return releases
}

// parseChangelogFile reads a local markdown changelog file and parses it.
func parseChangelogFile(path string) ([]sources.Release, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return parseChangelogContent(string(data)), nil
}

// fetchChangelogURL fetches a remote changelog via HTTP or HTTPS and parses it.
func fetchChangelogURL(rawURL string) ([]sources.Release, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: status %d", rawURL, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", rawURL, err)
	}
	return parseChangelogContent(string(body)), nil
}

// sortReleases sorts releases newest-first using ReleasedAt, falling back to
// CreatedAt. This ensures file-based and API-based entries are interleaved
// correctly after merging.
func sortReleases(releases []sources.Release) {
	sort.Slice(releases, func(i, j int) bool {
		di := releases[i].ReleasedAt
		if di.IsZero() {
			di = releases[i].CreatedAt
		}
		dj := releases[j].ReleasedAt
		if dj.IsZero() {
			dj = releases[j].CreatedAt
		}
		return di.After(dj)
	})
}

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
// a markdown link to the commit page on the source platform. Returns s
// unchanged when commitBaseURL is empty (e.g. local-only mode).
func linkifyCommits(s, commitBaseURL string) string {
	if commitBaseURL == "" {
		return s
	}
	return commitRef.ReplaceAllStringFunc(s, func(match string) string {
		hash := match[1 : len(match)-1] // strip surrounding parens
		return fmt.Sprintf("([%s](%s/%s))", hash, commitBaseURL, hash)
	})
}

func writeChangelog(releases []sources.Release, outputPath, commitBaseURL string) error {
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

// resolveSources validates the mode flag and determines which data sources to
// activate. It returns fetchAPI=true when the platform API should be queried
// and fetchLocal=true when -include sources should be loaded. An error is
// returned for any invalid combination of flags.
func resolveSources(mode, project string, includes []string) (fetchAPI, fetchLocal bool, err error) {
	switch mode {
	case "api", "local", "mixed":
	default:
		return false, false, fmt.Errorf("-mode must be one of: api, local, mixed")
	}
	fetchAPI = (mode == "api" || mode == "mixed") && project != ""
	fetchLocal = (mode == "local" || mode == "mixed") && len(includes) > 0
	if mode == "api" && project == "" {
		return false, false, fmt.Errorf("-mode api requires -project or $CI_PROJECT_PATH")
	}
	if mode == "local" && len(includes) == 0 {
		return false, false, fmt.Errorf("-mode local requires at least one -include source")
	}
	if !fetchAPI && !fetchLocal {
		return false, false, fmt.Errorf("no sources available; supply -project and/or -include")
	}
	return fetchAPI, fetchLocal, nil
}

// loadIncludes reads and merges releases from all -include sources. Each entry
// may be a local file path or an http:// / https:// URL.
func loadIncludes(paths []string) ([]sources.Release, error) {
	var all []sources.Release
	for _, path := range paths {
		var (
			releases []sources.Release
			err      error
		)
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			log.Printf("Fetching remote changelog: %s", path)
			releases, err = fetchChangelogURL(path)
		} else {
			log.Printf("Parsing local changelog: %s", path)
			releases, err = parseChangelogFile(path)
		}
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}
		log.Printf("Found %d release(s) in %s", len(releases), path)
		all = append(all, releases...)
	}
	return all, nil
}

func main() {
	var (
		flagURL     = flag.String("url", "", "GitLab instance URL (default: $CI_SERVER_URL or https://gitlab.com)")
		projectPath = flag.String("project", "", "GitLab project path or numeric ID (default: $CI_PROJECT_PATH)")
		token       = flag.String("token", "", "GitLab private token (or set GITLAB_TOKEN / CI_JOB_TOKEN env var)")
		output      = flag.String("output", "CHANGELOG.md", "Output file path")
		mode        = flag.String("mode", "mixed", `source mode: "api" (API only), "local" (-include sources only), "mixed" (both; default)`)
		includes    stringSlice
	)
	flag.Var(&includes, "include", "local file path or URL to merge into the output (repeatable)")
	flag.Parse()

	glSrc := sources.NewGitLabSourceFromEnv(*flagURL, *projectPath, *token)

	fetchAPI, fetchLocal, err := resolveSources(*mode, glSrc.Project(), includes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	var allReleases []sources.Release

	if fetchAPI {
		log.Printf("Fetching releases for project: %s", glSrc.Project())
		releases, err := glSrc.FetchReleases()
		if err != nil {
			log.Fatalf("Error fetching releases: %v", err)
		}
		log.Printf("Found %d release(s) from API", len(releases))
		allReleases = append(allReleases, releases...)
	}

	if fetchLocal {
		releases, err := loadIncludes(includes)
		if err != nil {
			log.Fatalf("Error loading includes: %v", err)
		}
		allReleases = append(allReleases, releases...)
	}

	sortReleases(allReleases)

	if err := writeChangelog(allReleases, *output, glSrc.CommitBaseURL()); err != nil {
		log.Fatalf("Error writing changelog: %v", err)
	}
	log.Printf("Changelog written to %s", *output)
}

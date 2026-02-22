package sources

import "time"

// Release represents a single changelog entry from any source.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ReleasedAt  time.Time `json:"released_at"`
}

// Source is implemented by any type that can provide a list of releases.
type Source interface {
	FetchReleases() ([]Release, error)
}

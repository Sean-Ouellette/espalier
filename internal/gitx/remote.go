package gitx

import (
	"net/url"
	"strings"
)

// ParseGitHubRemote extracts owner and repo from a GitHub remote URL.
// It accepts https://github.com/o/r[.git], ssh://git@github.com/o/r[.git],
// and the scp-like git@github.com:o/r[.git]. ok is false for any other host
// or shape.
func ParseGitHubRemote(raw string) (owner, repo string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}

	var host, path string
	if i := strings.Index(raw, "://"); i >= 0 {
		u, err := url.Parse(raw)
		if err != nil {
			return "", "", false
		}
		host, path = u.Hostname(), u.Path
	} else if at := strings.Index(raw, "@"); at >= 0 {
		// scp-like: user@host:path
		rest := raw[at+1:]
		colon := strings.Index(rest, ":")
		if colon < 0 {
			return "", "", false
		}
		host, path = rest[:colon], rest[colon+1:]
	} else {
		return "", "", false
	}

	if !strings.EqualFold(host, "github.com") {
		return "", "", false
	}

	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

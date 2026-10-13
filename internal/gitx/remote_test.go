package gitx

import "testing"

func TestParseGitHubRemote(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantOK    bool
	}{
		{"https", "https://github.com/Sean-Ouellette/espalier", "Sean-Ouellette", "espalier", true},
		{"https with .git", "https://github.com/Sean-Ouellette/espalier.git", "Sean-Ouellette", "espalier", true},
		{"https trailing slash", "https://github.com/Sean-Ouellette/espalier/", "Sean-Ouellette", "espalier", true},
		{"ssh scp-like", "git@github.com:Sean-Ouellette/espalier.git", "Sean-Ouellette", "espalier", true},
		{"ssh scp-like no .git", "git@github.com:Sean-Ouellette/espalier", "Sean-Ouellette", "espalier", true},
		{"ssh url", "ssh://git@github.com/Sean-Ouellette/espalier.git", "Sean-Ouellette", "espalier", true},
		{"surrounding whitespace", "  https://github.com/Sean-Ouellette/espalier\n", "Sean-Ouellette", "espalier", true},
		{"non-github host", "https://gitlab.com/Sean-Ouellette/espalier.git", "", "", false},
		{"missing repo", "https://github.com/Sean-Ouellette", "", "", false},
		{"empty", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := ParseGitHubRemote(tt.url)
			if owner != tt.wantOwner || repo != tt.wantRepo || ok != tt.wantOK {
				t.Errorf("ParseGitHubRemote(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.url, owner, repo, ok, tt.wantOwner, tt.wantRepo, tt.wantOK)
			}
		})
	}
}

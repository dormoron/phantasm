package change

import (
	"testing"
)

func TestParseGithubURL(t *testing.T) {
	tests := []struct {
		url   string
		owner string
		repo  string
	}{
		{
			url:   "https://github.com/dormoron/phantasm.git",
			owner: "dormoron",
			repo:  "phantasm",
		},
		{
			url:   "github.com/dormoron/phantasm",
			owner: "dormoron",
			repo:  "phantasm",
		},
		{
			url:   "git@github.com:dormoron/phantasm.git",
			owner: "github.com",
			repo:  "dormoron/phantasm",
		},
	}

	for _, test := range tests {
		owner, repo := ParseGithubURL(test.url)
		if owner != test.owner || repo != test.repo {
			t.Errorf("ParseGithubURL(%s) = (%s, %s), want (%s, %s)", test.url, owner, repo, test.owner, test.repo)
		}
	}
}

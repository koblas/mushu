package github

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

type PRValue struct {
	Number int
}

var pullURLRE = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/(\d+)(.*$)`)

func parsePRurl(prURL string) (*GitHubRepo, int, string, error) {
	if prURL == "" {
		return nil, 0, "", fmt.Errorf("invalid URL: %q", prURL)
	}

	u, err := url.Parse(prURL)
	if err != nil {
		return nil, 0, "", fmt.Errorf("parsing URL %q: %w", prURL, err)
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, 0, "", fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	m := pullURLRE.FindStringSubmatch(u.Path)
	if m == nil {
		return nil, 0, "", fmt.Errorf("not a pull request URL: %s", prURL)
	}

	repo := NewWithHost(m[1], m[2], u.Hostname())
	prNumber, _ := strconv.Atoi(m[3])
	tail := m[4]

	return repo, prNumber, tail, nil
}

func ParsePRValue(prValue string) (int, *GitHubRepo, error) {
	if repo, prNumber, _, err := parsePRurl(prValue); err == nil {
		return prNumber, repo, nil
	}

	// Try to parse as integer PR number
	num, err := strconv.Atoi(prValue)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid pull request value: %q", prValue)
	}

	return num, nil, nil
}

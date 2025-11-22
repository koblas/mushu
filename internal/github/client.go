package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/google/go-github/v75/github"
)

var ErrNoAuth = errors.New("authentication not successful")

type Config struct {
	Token string
	Repo  Repo
}

type Client struct {
	client *github.Client
	owner  string
	repo   string
}

func createGitHubClient(ctx context.Context, token string, repo Repo) (*github.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	host := repo.RepoHost()

	client := github.NewClient(nil)
	if token != "${GITHUB_TOKEN}" && token != "GITHUB_TOKEN" {
		client = client.WithAuthToken(token)
	}
	if host != "github.com" {
		url, err := url.Parse("https://" + host)
		if err != nil {
			return nil, fmt.Errorf("invalid hostname: %w", err)
		}
		ustr := url.String()
		client, err = client.WithEnterpriseURLs(ustr, ustr)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub base URL: %w", err)
		}
	}

	return client, nil
}

func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	// Create GitHub client
	slog.DebugContext(ctx, "Created GitHub client", "base_url", cfg.Repo.RepoHost())
	client, err := createGitHubClient(ctx, cfg.Token, cfg.Repo)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create GitHub client", "error", err)

		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return &Client{
		client: client,
		owner:  cfg.Repo.RepoOwner(),
		repo:   cfg.Repo.RepoName(),
	}, nil
}

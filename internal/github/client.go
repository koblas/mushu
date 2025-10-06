package github

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/v75/github"
	"github.com/koblas/mushu/internal/logging"
)

type Config struct {
	Token   string
	BaseURL string
	Owner   string
	Repo    string
}

type Client struct {
	client *github.Client
	owner  string
	repo   string
}

func createGitHubClient(token string, baseURL string) (*github.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	client := github.NewClient(nil)

	if baseURL != "" && baseURL != "https://api.github.com" {
		var err error
		client.BaseURL, err = url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub base URL: %w", err)
		}
	}

	return client, nil
}

func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	// Create GitHub client
	logging.Debug(ctx, "Created GitHub client", "base_url", cfg.BaseURL)
	client, err := createGitHubClient(cfg.Token, cfg.BaseURL)
	if err != nil {
		logging.Error(ctx, "Failed to create GitHub client", "error", err)

		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// // Create team service
	// yamlService := teams.NewYAMLTeamService(os.DirFS("."), cfg.Teams.TeamsFile, cfg.Teams.TeamsDir)
	// // Load teams (optional - only fail on real errors, not missing files)
	// if err := yamlService.Load(ctx); err != nil && !errors.Is(err, os.ErrNotExist) {
	// 	logging.Error(ctx, "Failed to load teams from YAML", "error", err)

	// 	return nil, fmt.Errorf("failed to load teams: %w", err)
	// }

	// var teamService teams.TeamService = yamlService
	// if cfg.Teams.UseGitHubAPI {
	// 	logging.Debug(ctx, "Using GitHub API for team management")
	// 	githubTeamService := teams.NewGitHubTeamService(client, cfg.GitHub.Owner, cfg.GitHub.Repo)
	// 	teamService = teams.NewCompositeTeamService(yamlService, githubTeamService, true)
	// } else {
	// 	logging.Debug(ctx, "Using YAML-only team management")
	// }

	return &Client{
		client: client,
		owner:  cfg.Owner,
		repo:   cfg.Repo,
	}, nil
}

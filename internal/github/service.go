package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/go-github/v60/github"
	"github.com/koblas/mushu/internal/policy"
	"github.com/koblas/mushu/internal/rules"
	"github.com/koblas/mushu/internal/teams"
)

// GitHubService handles GitHub API integration
type GitHubService struct {
	client      *github.Client
	owner       string
	repo        string
	teamService teams.TeamService
}

// NewGitHubService creates a new GitHub service
func NewGitHubService(client *github.Client, owner, repo string, teamService teams.TeamService) *GitHubService {
	return &GitHubService{
		client:      client,
		owner:       owner,
		repo:        repo,
		teamService: teamService,
	}
}

// GetPRData fetches pull request data from GitHub
func (gs *GitHubService) GetPRData(ctx context.Context, prNumber int) (*policy.PRData, error) {
	// Get PR details
	pr, _, err := gs.client.PullRequests.Get(ctx, gs.owner, gs.repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR %d: %w", prNumber, err)
	}

	// Get PR files
	files, _, err := gs.client.PullRequests.ListFiles(ctx, gs.owner, gs.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR files: %w", err)
	}

	// Get PR reviews
	reviews, _, err := gs.client.PullRequests.ListReviews(ctx, gs.owner, gs.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR reviews: %w", err)
	}

	// Convert to internal format
	prData := &policy.PRData{
		Number:       prNumber,
		Title:        pr.GetTitle(),
		State:        pr.GetState(),
		Author:       pr.GetUser().GetLogin(),
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
		Labels:       gs.extractLabels(pr.Labels),
		Files:        gs.convertFiles(files),
		Reviews:      gs.convertReviews(reviews),
	}

	// Get reviewers
	prData.Reviewers = gs.extractReviewers(reviews)
	prData.Approvals = gs.countApprovals(reviews)

	return prData, nil
}

// extractLabels extracts label names from GitHub labels
func (gs *GitHubService) extractLabels(labels []*github.Label) []string {
	var names []string
	for _, label := range labels {
		names = append(names, label.GetName())
	}
	return names
}

// convertFiles converts GitHub files to internal format
func (gs *GitHubService) convertFiles(files []*github.CommitFile) []rules.PRFile {
	var prFiles []rules.PRFile
	for _, file := range files {
		prFiles = append(prFiles, rules.PRFile{
			Filename:  file.GetFilename(),
			Status:    file.GetStatus(),
			Additions: file.GetAdditions(),
			Deletions: file.GetDeletions(),
			Changes:   file.GetChanges(),
		})
	}
	return prFiles
}

// convertReviews converts GitHub reviews to internal format
func (gs *GitHubService) convertReviews(reviews []*github.PullRequestReview) []policy.Review {
	var prReviews []policy.Review
	for _, review := range reviews {
		// Get reviewer teams
		reviewerTeams, err := gs.teamService.GetUserTeams(context.Background(), review.GetUser().GetLogin())
		if err != nil {
			// If we can't get teams, continue without them
			reviewerTeams = []string{}
		}

		prReviews = append(prReviews, policy.Review{
			Reviewer:      review.GetUser().GetLogin(),
			ReviewerTeams: reviewerTeams,
			State:         review.GetState(),
			SubmittedAt:   review.GetSubmittedAt().Format("2006-01-02T15:04:05Z"),
		})
	}
	return prReviews
}

// extractReviewers extracts unique reviewer usernames
func (gs *GitHubService) extractReviewers(reviews []*github.PullRequestReview) []string {
	reviewerMap := make(map[string]bool)
	for _, review := range reviews {
		reviewerMap[review.GetUser().GetLogin()] = true
	}

	var reviewers []string
	for reviewer := range reviewerMap {
		reviewers = append(reviewers, reviewer)
	}
	return reviewers
}

// countApprovals counts the number of approved reviews
func (gs *GitHubService) countApprovals(reviews []*github.PullRequestReview) int {
	count := 0
	for _, review := range reviews {
		if review.GetState() == "APPROVED" {
			count++
		}
	}
	return count
}

// ValidatePR validates a pull request against policies
func (gs *GitHubService) ValidatePR(ctx context.Context, prNumber int, policyEngine *policy.PolicyEngine) (*policy.EvaluationResult, error) {
	// Get PR data
	prData, err := gs.GetPRData(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR data: %w", err)
	}

	// Evaluate against policies
	result, err := policyEngine.EvaluatePR(ctx, prData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate PR: %w", err)
	}

	return result, nil
}

// CreateGitHubClient creates a GitHub client with authentication
func CreateGitHubClient(token string, baseURL string) (*github.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	if baseURL != "" && baseURL != "https://api.github.com" {
		var err error
		client.BaseURL, err = url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub base URL: %w", err)
		}
	}

	return client, nil
}

// ParsePRNumber parses a PR number from string
func ParsePRNumber(prStr string) (int, error) {
	prNumber, err := strconv.Atoi(prStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PR number: %s", prStr)
	}
	return prNumber, nil
}

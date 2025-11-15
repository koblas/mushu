package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v75/github"
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
func (gs *Client) GetPRData(ctx context.Context, prNumber int, lookup teams.Lookup) (*PRData, error) {
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

	annotated, err := gs.convertReviews(ctx, reviews, lookup)
	if err != nil {
		return nil, fmt.Errorf("failed to convert PR reviews: %w", err)
	}

	// Convert to internal format
	prData := &PRData{
		Number:       prNumber,
		Title:        pr.GetTitle(),
		State:        pr.GetState(),
		Author:       pr.GetUser().GetLogin(),
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
		Labels:       gs.extractLabels(pr.Labels),
		Files:        gs.convertFiles(files),
		Reviews:      annotated,
	}

	// Get reviewers
	prData.Reviewers = gs.extractReviewers(reviews)
	prData.Approvals = gs.countApprovals(reviews)

	return prData, nil
}

// extractLabels extracts label names from GitHub labels
func (gs *Client) extractLabels(labels []*github.Label) []string {
	var names []string
	for _, label := range labels {
		names = append(names, label.GetName())
	}
	return names
}

// convertFiles converts GitHub files to internal format
func (gs *Client) convertFiles(files []*github.CommitFile) []rules.PRFile {
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
func (gs *Client) convertReviews(ctx context.Context, reviews []*github.PullRequestReview, lookup teams.Lookup) ([]Review, error) {
	users := map[string]struct{}{}
	for _, review := range reviews {
		users[review.GetUser().GetLogin()] = struct{}{}
	}

	// now lookup the teams for each user
	userToTeams := map[string][]string{}
	for user := range users {
		teams, err := lookup.GetUserTeams(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to get user teams: %w", err)
		}
		userToTeams[user] = teams
	}

	var prReviews []Review
	for _, review := range reviews {
		prReviews = append(prReviews, Review{
			Reviewer:      review.GetUser().GetLogin(),
			ReviewerTeams: userToTeams[review.GetUser().GetLogin()],
			State:         review.GetState(),
			SubmittedAt:   review.GetSubmittedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	return prReviews, nil
}

// extractReviewers extracts unique reviewer usernames
func (gs *Client) extractReviewers(reviews []*github.PullRequestReview) []string {
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
func (gs *Client) countApprovals(reviews []*github.PullRequestReview) int {
	count := 0
	for _, review := range reviews {
		if review.GetState() == "APPROVED" {
			count++
		}
	}
	return count
}

// ParsePRNumber parses a PR number from string
func ParsePRNumber(prStr string) (int, error) {
	prNumber, err := strconv.Atoi(prStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PR number: %s", prStr)
	}
	return prNumber, nil
}

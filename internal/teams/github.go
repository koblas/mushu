package teams

import (
	"context"
	"fmt"

	"github.com/google/go-github/v79/github"
)

// GitHubTeamService implements TeamService using GitHub API
type GitHubTeamService struct {
	client *github.Client
	owner  string
	repo   string
}

// NewGitHubTeamService creates a new GitHub-based team service
func NewGitHubTeamService(client *github.Client, owner, repo string) *GitHubTeamService {
	return &GitHubTeamService{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// GetUserTeams returns team memberships for a user from GitHub
func (s *GitHubTeamService) GetUserTeams(ctx context.Context, username string) ([]string, error) {
	// Get user's organization memberships
	orgs, _, err := s.client.Organizations.List(ctx, username, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	var teams []string
	for _, org := range orgs {
		if *org.Login == s.owner {
			// Get user's teams in this organization
			userTeams, _, err := s.client.Teams.ListUserTeams(ctx, nil)
			if err != nil {
				continue // Skip if we can't get teams for this org
			}

			for _, team := range userTeams {
				if team.Organization != nil && *team.Organization.Login == s.owner {
					teams = append(teams, *team.Slug)
				}
			}
		}
	}

	return teams, nil
}

// GetTeamMembers returns all members of a team from GitHub
func (s *GitHubTeamService) GetTeamMembers(ctx context.Context, teamSlug string) ([]string, error) {
	// First, get the team by slug
	team, _, err := s.client.Teams.GetTeamBySlug(ctx, s.owner, teamSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get team %s: %w", teamSlug, err)
	}

	// Get team members
	members, _, err := s.client.Teams.ListTeamMembersByID(ctx, *team.Organization.ID, *team.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	var usernames []string
	for _, member := range members {
		usernames = append(usernames, *member.Login)
	}

	return usernames, nil
}

// GetUserRoles is an alias for GetUserTeams for backward compatibility
func (s *GitHubTeamService) GetUserRoles(ctx context.Context, username string) ([]string, error) {
	return s.GetUserTeams(ctx, username)
}

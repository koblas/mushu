package teams

import (
	"context"
)

// CompositeTeamService combines multiple team services with fallback logic
type CompositeTeamService struct {
	yamlService   *YAMLTeamService
	githubService *GitHubTeamService
	useGitHubAPI  bool
}

// NewCompositeTeamService creates a composite team service
func NewCompositeTeamService(yamlService *YAMLTeamService, githubService *GitHubTeamService, useGitHubAPI bool) *CompositeTeamService {
	return &CompositeTeamService{
		yamlService:   yamlService,
		githubService: githubService,
		useGitHubAPI:  useGitHubAPI,
	}
}

// GetUserTeams returns team memberships with fallback logic
func (s *CompositeTeamService) GetUserTeams(ctx context.Context, username string) ([]string, error) {
	// Try YAML first
	teams, err := s.yamlService.GetUserTeams(ctx, username)
	if err == nil && len(teams) > 0 {
		return teams, nil
	}

	// Fallback to GitHub API if enabled and YAML didn't have results
	if s.useGitHubAPI && s.githubService != nil {
		githubTeams, err := s.githubService.GetUserTeams(ctx, username)
		if err != nil {
			// If GitHub fails, return YAML results (even if empty)
			return teams, nil //nolint:nilerr
		}

		return githubTeams, nil
	}

	return teams, nil
}

// GetTeamMembers returns team members with fallback logic
func (s *CompositeTeamService) GetTeamMembers(ctx context.Context, teamSlug string) ([]string, error) {
	// Try YAML first
	members, err := s.yamlService.GetTeamMembers(ctx, teamSlug)
	if err == nil && len(members) > 0 {
		return members, nil
	}

	// Fallback to GitHub API if enabled and YAML didn't have results
	if s.useGitHubAPI && s.githubService != nil {
		githubMembers, err := s.githubService.GetTeamMembers(ctx, teamSlug)
		if err != nil {
			// If GitHub fails, return YAML results (even if empty)
			return members, nil //nolint:nilerr
		}

		return githubMembers, nil
	}

	return members, nil
}

// GetUserRoles is an alias for GetUserTeams for backward compatibility
func (s *CompositeTeamService) GetUserRoles(ctx context.Context, username string) ([]string, error) {
	return s.GetUserTeams(ctx, username)
}

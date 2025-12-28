package teams

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// TeamService defines the interface for team management
type TeamService interface {
	// GetUserTeams returns team memberships for an external user (GitHub username)
	GetUserTeams(ctx context.Context, username string) ([]string, error)
	// GetTeamMembers returns all members of a team (external usernames)
	GetTeamMembers(ctx context.Context, teamSlug string) ([]string, error)
	// GetUserRoles is an alias for GetUserTeams for backward compatibility
	GetUserRoles(ctx context.Context, username string) ([]string, error)
}

// TeamsData represents the structure of teams.yaml
type TeamsData struct {
	Teams map[string]Team `yaml:"teams"`
}

// Team represents a team definition
type Team struct {
	Description string   `yaml:"description"`
	Members     []string `yaml:"members"`
}

// YAMLTeamService implements TeamService using YAML files
type YAMLTeamService struct {
	fs        fs.FS
	teamsFile string
	teams     map[string]Team
}

// NewYAMLTeamService creates a new YAML-based team service
func NewYAMLTeamService(fsys fs.FS, teamsFile string) *YAMLTeamService {
	return &YAMLTeamService{
		fs:        fsys,
		teamsFile: teamsFile,
		teams:     make(map[string]Team),
	}
}

// Load loads team data from YAML files (optional)
func (s *YAMLTeamService) Load(ctx context.Context) error {
	err := doublestar.GlobWalk(s.fs, s.teamsFile, func(path string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil
		}

		return s.loadTeamsFile(ctx, path)
	})
	if err != nil {
		return fmt.Errorf("failed to load teams file: %w", err)
	}

	return nil
}

// loadTeamsFile loads a single teams YAML file
func (s *YAMLTeamService) loadTeamsFile(ctx context.Context, filename string) error {
	data, err := fs.ReadFile(s.fs, filename)
	if err != nil {
		// We don't want to fail if the file doesn't exist or is invalid
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
			return nil
		}

		return fmt.Errorf("failed to read teams file: %w", err)
	}

	var teamsData TeamsData
	if err := yaml.Unmarshal(data, &teamsData); err != nil {
		return fmt.Errorf("failed to parse teams file: %w", err)
	}

	maps.Copy(s.teams, teamsData.Teams)

	slog.InfoContext(ctx, "Successfully loaded teams file", slog.String("teams_file", filename))

	return nil
}

// GetUserTeams returns team memberships for a user
func (s *YAMLTeamService) GetUserTeams(ctx context.Context, username string) ([]string, error) {
	var teams []string

	for teamName, team := range s.teams {
		for _, member := range team.Members {
			if member == username {
				teams = append(teams, teamName)
				break
			}
		}
	}

	return teams, nil
}

// GetTeamMembers returns all members of a team
func (s *YAMLTeamService) GetTeamMembers(ctx context.Context, teamSlug string) ([]string, error) {
	team, exists := s.teams[teamSlug]
	if !exists {
		return nil, fmt.Errorf("team %s not found", teamSlug)
	}

	return team.Members, nil
}

// GetUserRoles is an alias for GetUserTeams for backward compatibility
func (s *YAMLTeamService) GetUserRoles(ctx context.Context, username string) ([]string, error) {
	return s.GetUserTeams(ctx, username)
}

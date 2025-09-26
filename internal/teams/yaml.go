package teams

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	teamsFile string
	teamsDir  string
	teams     map[string]Team
}

// NewYAMLTeamService creates a new YAML-based team service
func NewYAMLTeamService(teamsFile, teamsDir string) *YAMLTeamService {
	return &YAMLTeamService{
		teamsFile: teamsFile,
		teamsDir:  teamsDir,
		teams:     make(map[string]Team),
	}
}

// Load loads team data from YAML files
func (s *YAMLTeamService) Load(ctx context.Context) error {
	// Load main teams file
	if err := s.loadTeamsFile(s.teamsFile); err != nil {
		return fmt.Errorf("failed to load teams file %s: %w", s.teamsFile, err)
	}

	// Load additional team files from teams directory
	if s.teamsDir != "" {
		if err := s.loadTeamsDirectory(s.teamsDir); err != nil {
			return fmt.Errorf("failed to load teams directory %s: %w", s.teamsDir, err)
		}
	}

	return nil
}

// loadTeamsFile loads a single teams YAML file
func (s *YAMLTeamService) loadTeamsFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var teamsData TeamsData
	if err := yaml.Unmarshal(data, &teamsData); err != nil {
		return fmt.Errorf("failed to parse teams file: %w", err)
	}

	// Merge teams into the main map
	for name, team := range teamsData.Teams {
		s.teams[name] = team
	}

	return nil
}

// loadTeamsDirectory loads all YAML files from a directory
func (s *YAMLTeamService) loadTeamsDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		if err := s.loadTeamsFile(filename); err != nil {
			return fmt.Errorf("failed to load team file %s: %w", filename, err)
		}
	}

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

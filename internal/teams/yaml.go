package teams

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
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
	fs        fs.FS
	teamsFile string
	teamsDir  string
	teams     map[string]Team
}

// NewYAMLTeamService creates a new YAML-based team service
func NewYAMLTeamService(fsys fs.FS, teamsFile, teamsDir string) *YAMLTeamService {
	return &YAMLTeamService{
		fs:        fsys,
		teamsFile: teamsFile,
		teamsDir:  teamsDir,
		teams:     make(map[string]Team),
	}
}

// Load loads team data from YAML files (optional)
func (s *YAMLTeamService) Load(ctx context.Context) error {
	// Load main teams file (optional)
	if s.teamsFile != "" {
		if err := s.loadTeamsFile(ctx, s.teamsFile); err != nil {
			// Log at info level but don't fail - team loading is optional
			slog.InfoContext(ctx, "Teams file not found or failed to load", "teams_file", s.teamsFile, "error", err)

			return fmt.Errorf("failed to load teams file: %w", err)
		}
	}

	// Load additional team files from teams directory (optional)
	if s.teamsDir != "" {
		if err := s.loadTeamsDirectory(ctx, s.teamsDir); err != nil {
			// Log at info level but don't fail - team loading is optional
			slog.InfoContext(ctx, "Teams directory not found or failed to load", "teams_dir", s.teamsDir, "error", err)

			return fmt.Errorf("failed to load teams directory: %w", err)
		}
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

	slog.InfoContext(ctx, "Successfully loaded teams file", "teams_file", filename)

	return nil
}

// loadTeamsDirectory loads all YAML files from a directory
func (s *YAMLTeamService) loadTeamsDirectory(ctx context.Context, dir string) error {
	entries, err := fs.ReadDir(s.fs, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
			return nil
		}

		return fmt.Errorf("failed to read teams directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		if err := s.loadTeamsFile(ctx, filename); err != nil {
			return fmt.Errorf("directory team file %s: %w", filename, err)
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

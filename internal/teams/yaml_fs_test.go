package teams

import (
	"context"
	"io/fs"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLTeamService_Load(t *testing.T) {
	tests := []struct {
		name      string
		fsys      fs.FS
		teamsFile string
		teamsDir  string
		wantTeams map[string]Team
		wantError bool
		errorType string
	}{
		{
			name: "load single teams file",
			fsys: fstest.MapFS{
				"teams.yaml": &fstest.MapFile{
					Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
  backend:
    description: "Backend developers"
    members:
      - charlie
      - diana
`),
				},
			},
			teamsFile: "teams.yaml",
			teamsDir:  "",
			wantTeams: map[string]Team{
				"frontend": {
					Description: "Frontend developers",
					Members:     []string{"alice", "bob"},
				},
				"backend": {
					Description: "Backend developers",
					Members:     []string{"charlie", "diana"},
				},
			},
			wantError: false,
		},
		{
			name: "load teams directory",
			fsys: fstest.MapFS{
				"teams/frontend.yaml": &fstest.MapFile{
					Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
`),
				},
				"teams/backend.yaml": &fstest.MapFile{
					Data: []byte(`
teams:
  backend:
    description: "Backend developers"
    members:
      - charlie
      - diana
`),
				},
			},
			teamsFile: "",
			teamsDir:  "teams",
			wantTeams: map[string]Team{
				"frontend": {
					Description: "Frontend developers",
					Members:     []string{"alice", "bob"},
				},
				"backend": {
					Description: "Backend developers",
					Members:     []string{"charlie", "diana"},
				},
			},
			wantError: false,
		},
		{
			name: "load both teams file and directory",
			fsys: fstest.MapFS{
				"teams.yaml": &fstest.MapFile{
					Data: []byte(`
teams:
  main:
    description: "Main team"
    members:
      - admin
`),
				},
				"teams/frontend.yaml": &fstest.MapFile{
					Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
`),
				},
			},
			teamsFile: "teams.yaml",
			teamsDir:  "teams",
			wantTeams: map[string]Team{
				"main": {
					Description: "Main team",
					Members:     []string{"admin"},
				},
				"frontend": {
					Description: "Frontend developers",
					Members:     []string{"alice", "bob"},
				},
			},
			wantError: false,
		},
		{
			name:      "missing teams file - should not error",
			fsys:      fstest.MapFS{},
			teamsFile: "nonexistent.yaml",
			teamsDir:  "",
			wantTeams: map[string]Team{},
			wantError: false,
		},
		{
			name:      "missing teams directory - should not error",
			fsys:      fstest.MapFS{},
			teamsFile: "",
			teamsDir:  "nonexistent",
			wantTeams: map[string]Team{},
			wantError: false,
		},
		{
			name: "malformed YAML - should error",
			fsys: fstest.MapFS{
				"teams.yaml": &fstest.MapFile{
					Data: []byte(`invalid: [unclosed`),
				},
			},
			teamsFile: "teams.yaml",
			teamsDir:  "",
			wantTeams: map[string]Team{},
			wantError: true,
			errorType: "parse",
		},
		{
			name: "empty teams file",
			fsys: fstest.MapFS{
				"teams.yaml": &fstest.MapFile{
					Data: []byte(`teams: {}`),
				},
			},
			teamsFile: "teams.yaml",
			teamsDir:  "",
			wantTeams: map[string]Team{},
			wantError: false,
		},
		{
			name: "teams file with no teams key",
			fsys: fstest.MapFS{
				"teams.yaml": &fstest.MapFile{
					Data: []byte(`other: value`),
				},
			},
			teamsFile: "teams.yaml",
			teamsDir:  "",
			wantTeams: map[string]Team{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewYAMLTeamService(tt.fsys, tt.teamsFile, tt.teamsDir)

			err := service.Load(t.Context())

			if tt.wantError {
				require.Error(t, err)
				if tt.errorType == "parse" {
					assert.Contains(t, err.Error(), "failed to parse teams file")
				}
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantTeams, service.teams)
		})
	}
}

func TestYAMLTeamService_GetUserTeams(t *testing.T) {
	fsys := fstest.MapFS{
		"teams.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
  backend:
    description: "Backend developers"
    members:
      - charlie
      - alice
  devops:
    description: "DevOps team"
    members:
      - diana
`),
		},
	}

	service := NewYAMLTeamService(fsys, "teams.yaml", "")
	err := service.Load(context.Background())
	require.NoError(t, err)

	tests := []struct {
		username string
		want     []string
	}{
		{
			username: "alice",
			want:     []string{"frontend", "backend"},
		},
		{
			username: "bob",
			want:     []string{"frontend"},
		},
		{
			username: "charlie",
			want:     []string{"backend"},
		},
		{
			username: "diana",
			want:     []string{"devops"},
		},
		{
			username: "nonexistent",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			teams, err := service.GetUserTeams(context.Background(), tt.username)
			require.NoError(t, err)

			// Sort both slices for consistent comparison
			sort.Strings(teams)
			sort.Strings(tt.want)

			assert.Equal(t, tt.want, teams)
		})
	}
}

func TestYAMLTeamService_GetTeamMembers(t *testing.T) {
	fsys := fstest.MapFS{
		"teams.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
  backend:
    description: "Backend developers"
    members:
      - charlie
      - diana
`),
		},
	}

	service := NewYAMLTeamService(fsys, "teams.yaml", "")
	err := service.Load(context.Background())
	require.NoError(t, err)

	tests := []struct {
		teamSlug string
		want     []string
		wantErr  bool
	}{
		{
			teamSlug: "frontend",
			want:     []string{"alice", "bob"},
			wantErr:  false,
		},
		{
			teamSlug: "backend",
			want:     []string{"charlie", "diana"},
			wantErr:  false,
		},
		{
			teamSlug: "nonexistent",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.teamSlug, func(t *testing.T) {
			members, err := service.GetTeamMembers(context.Background(), tt.teamSlug)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, members)
			}
		})
	}
}

func TestYAMLTeamService_GetUserRoles(t *testing.T) {
	fsys := fstest.MapFS{
		"teams.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
`),
		},
	}

	service := NewYAMLTeamService(fsys, "teams.yaml", "")
	err := service.Load(context.Background())
	require.NoError(t, err)

	// GetUserRoles should be an alias for GetUserTeams
	teams, err := service.GetUserRoles(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, []string{"frontend"}, teams)
}

func TestYAMLTeamService_LoadWithMultipleFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"teams.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  main:
    description: "Main team"
    members:
      - admin
`),
		},
		"teams/frontend.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  frontend:
    description: "Frontend developers"
    members:
      - alice
      - bob
`),
		},
		"teams/backend.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  backend:
    description: "Backend developers"
    members:
      - charlie
      - diana
`),
		},
		"teams/ignored.txt": &fstest.MapFile{
			Data: []byte(`This should be ignored`),
		},
	}

	service := NewYAMLTeamService(fsys, "teams.yaml", "teams")
	err := service.Load(context.Background())
	require.NoError(t, err)

	// Should have teams from both main file and directory
	expectedTeams := map[string]Team{
		"main": {
			Description: "Main team",
			Members:     []string{"admin"},
		},
		"frontend": {
			Description: "Frontend developers",
			Members:     []string{"alice", "bob"},
		},
		"backend": {
			Description: "Backend developers",
			Members:     []string{"charlie", "diana"},
		},
	}

	assert.Equal(t, expectedTeams, service.teams)
}

func TestYAMLTeamService_LoadWithOverlappingTeams(t *testing.T) {
	fsys := fstest.MapFS{
		"teams.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  shared:
    description: "Shared team from main file"
    members:
      - alice
      - bob
`),
		},
		"teams/shared.yaml": &fstest.MapFile{
			Data: []byte(`
teams:
  shared:
    description: "Shared team from directory"
    members:
      - charlie
      - diana
`),
		},
	}

	service := NewYAMLTeamService(fsys, "teams.yaml", "teams")
	err := service.Load(context.Background())
	require.NoError(t, err)

	// Directory files should override main file (loaded later)
	expectedTeam := Team{
		Description: "Shared team from directory",
		Members:     []string{"charlie", "diana"},
	}

	assert.Equal(t, expectedTeam, service.teams["shared"])
}

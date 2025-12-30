package teams_test

import (
	"context"
	"os"
	"testing"

	"github.com/koblas/mushu/internal/teams"
)

func TestYAMLTeamService(t *testing.T) {
	service := teams.NewYAMLTeamService(os.DirFS("."), "testdata/teams.yaml")

	ctx := context.Background()
	err := service.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load teams: %v", err)
	}

	// Test getting user teams
	teams, err := service.GetUserTeams(ctx, "alice")
	if err != nil {
		t.Fatalf("Failed to get user teams: %v", err)
	}

	expectedTeams := []string{"senior-backend", "platform-team", "database-team"}
	if len(teams) != len(expectedTeams) {
		t.Errorf("Expected %d teams, got %d", len(expectedTeams), len(teams))
	}

	// Test getting team members
	members, err := service.GetTeamMembers(ctx, "security-team")
	if err != nil {
		t.Fatalf("Failed to get team members: %v", err)
	}

	expectedMembers := []string{"bob", "eve"}
	if len(members) != len(expectedMembers) {
		t.Errorf("Expected %d members, got %d", len(expectedMembers), len(members))
	}
}

func TestCompositeTeamService(t *testing.T) {
	yamlService := teams.NewYAMLTeamService(os.DirFS("."), "testdata/teams.yaml")
	compositeService := teams.NewCompositeTeamService(yamlService, nil, false)

	ctx := context.Background()
	err := yamlService.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load teams: %v", err)
	}

	// Test composite service
	teams, err := compositeService.GetUserTeams(ctx, "alice")
	if err != nil {
		t.Fatalf("Failed to get user teams: %v", err)
	}

	if len(teams) == 0 {
		t.Error("Expected teams to be returned")
	}
}

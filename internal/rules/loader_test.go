package rules_test

import (
	"context"
	"embed"
	"testing"

	"github.com/koblas/mushu/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testfs embed.FS

func TestRuleLoader(t *testing.T) {
	loader := rules.NewRuleLoader("testdata",
		rules.WithFileSystem(testfs),
		rules.WithFilename("parse.yaml"),
	)

	rules, err := loader.LoadRules(t.Context())
	require.NoError(t, err, "Failed to load rules")

	require.NotEmpty(t, rules, "Expected at least one rule to be loaded")

	// Check that we have the expected rules
	var ruleNames []string
	for _, rule := range rules {
		ruleNames = append(ruleNames, rule.Name)
	}

	expectedRules := []string{"security-review", "large-changes", "infrastructure", "binary-files"}
	assert.ElementsMatch(t, ruleNames, expectedRules, "Loaded rules do not match expected rules")
}

// mockPRData implements the rules.PRData interface for testing
type mockPRData struct {
	author       string
	sourceBranch string
	targetBranch string
	files        []rules.PRFile
}

func (m *mockPRData) GetAuthor() string {
	return m.author
}

func (m *mockPRData) GetSourceBranch() string {
	return m.sourceBranch
}

func (m *mockPRData) GetTargetBranch() string {
	return m.targetBranch
}

func (m *mockPRData) GetFiles() []rules.PRFile {
	return m.files
}

// mockTeamLookup implements the rules.TeamLookup interface for testing
type mockTeamLookup struct {
	teams map[string][]string
	err   error
}

func (m *mockTeamLookup) GetUserTeams(ctx context.Context, username string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.teams[username], nil
}

func TestMatchWhen_NoCondition(t *testing.T) {
	rule := &rules.Rule{
		Name: "test-rule",
		When: nil,
	}

	prData := &mockPRData{
		author: "alice",
	}

	match, err := rule.MatchWhen(t.Context(), prData, nil)
	require.NoError(t, err)
	assert.True(t, match, "rule with no when condition should always match")
}

func TestMatchWhen_BranchConditions(t *testing.T) {
	tests := []struct {
		name        string
		when        *rules.RuleWhen
		prData      *mockPRData
		expectMatch bool
		expectError bool
	}{
		{
			name: "target branch matches",
			when: &rules.RuleWhen{
				Branch: &rules.RuleWhenBranch{
					Target: []string{"main", "develop"},
				},
			},
			prData: &mockPRData{
				targetBranch: "main",
			},
			expectMatch: true,
		},
		{
			name: "target branch does not match",
			when: &rules.RuleWhen{
				Branch: &rules.RuleWhenBranch{
					Target: []string{"main", "develop"},
				},
			},
			prData: &mockPRData{
				targetBranch: "feature-x",
			},
			expectMatch: false,
		},
		{
			name: "source branch matches wildcard",
			when: &rules.RuleWhen{
				Branch: &rules.RuleWhenBranch{
					Source: []string{"feature/*", "bugfix/*"},
				},
			},
			prData: &mockPRData{
				sourceBranch: "feature/new-feature",
			},
			expectMatch: true,
		},
		{
			name: "source branch does not match",
			when: &rules.RuleWhen{
				Branch: &rules.RuleWhenBranch{
					Source: []string{"feature/*"},
				},
			},
			prData: &mockPRData{
				sourceBranch: "bugfix/fix-bug",
			},
			expectMatch: false,
		},
		{
			name: "both source and target match",
			when: &rules.RuleWhen{
				Branch: &rules.RuleWhenBranch{
					Target: []string{"main"},
					Source: []string{"feature/*"},
				},
			},
			prData: &mockPRData{
				targetBranch: "main",
				sourceBranch: "feature/test",
			},
			expectMatch: true,
		},
		{
			name: "target matches but source does not",
			when: &rules.RuleWhen{
				Branch: &rules.RuleWhenBranch{
					Target: []string{"main"},
					Source: []string{"feature/*"},
				},
			},
			prData: &mockPRData{
				targetBranch: "main",
				sourceBranch: "hotfix/urgent",
			},
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &rules.Rule{
				Name: "test-rule",
				When: tt.when,
			}

			match, err := rule.MatchWhen(t.Context(), tt.prData, nil)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectMatch, match)
			}
		})
	}
}

func TestMatchWhen_FileConditions(t *testing.T) {
	tests := []struct {
		name        string
		when        *rules.RuleWhen
		files       []rules.PRFile
		expectMatch bool
	}{
		{
			name: "files match include pattern",
			when: &rules.RuleWhen{
				File: &rules.RuleWhenFile{
					Include: []string{"src/**"},
				},
			},
			files: []rules.PRFile{
				{Filename: "src/main.go"},
				{Filename: "src/api/handler.go"},
			},
			expectMatch: true,
		},
		{
			name: "files do not match include pattern",
			when: &rules.RuleWhen{
				File: &rules.RuleWhenFile{
					Include: []string{"src/**"},
				},
			},
			files: []rules.PRFile{
				{Filename: "docs/README.md"},
			},
			expectMatch: false,
		},
		{
			name: "files match but excluded",
			when: &rules.RuleWhen{
				File: &rules.RuleWhenFile{
					Include: []string{"src/**"},
					Exclude: []string{"src/experimental/**"},
				},
			},
			files: []rules.PRFile{
				{Filename: "src/experimental/test.go"},
			},
			expectMatch: false,
		},
		{
			name: "some files excluded, some remain",
			when: &rules.RuleWhen{
				File: &rules.RuleWhenFile{
					Include: []string{"src/**"},
					Exclude: []string{"src/experimental/**"},
				},
			},
			files: []rules.PRFile{
				{Filename: "src/main.go"},
				{Filename: "src/experimental/test.go"},
			},
			expectMatch: true,
		},
		{
			name: "no include pattern matches all",
			when: &rules.RuleWhen{
				File: &rules.RuleWhenFile{
					Exclude: []string{"*.md"},
				},
			},
			files: []rules.PRFile{
				{Filename: "src/main.go"},
			},
			expectMatch: true,
		},
		{
			name: "no include pattern but all excluded",
			when: &rules.RuleWhen{
				File: &rules.RuleWhenFile{
					Exclude: []string{"*.md"},
				},
			},
			files: []rules.PRFile{
				{Filename: "README.md"},
			},
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &rules.Rule{
				Name: "test-rule",
				When: tt.when,
			}

			prData := &mockPRData{
				files: tt.files,
			}

			match, err := rule.MatchWhen(t.Context(), prData, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expectMatch, match)
		})
	}
}

func TestMatchWhen_ContributorConditions(t *testing.T) {
	teamLookup := &mockTeamLookup{
		teams: map[string][]string{
			"alice": {"dev-team", "frontend-team"},
			"bob":   {"dev-team", "backend-team"},
			"carol": {"qa-team"},
		},
	}

	tests := []struct {
		name        string
		when        *rules.RuleWhen
		author      string
		teamLookup  rules.TeamLookup
		expectMatch bool
		expectError bool
	}{
		{
			name: "user in users list",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Users: []string{"alice", "bob"},
				},
			},
			author:      "alice",
			expectMatch: true,
		},
		{
			name: "user not in users list",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Users: []string{"alice", "bob"},
				},
			},
			author:      "carol",
			expectMatch: false,
		},
		{
			name: "user in team",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Teams: []string{"dev-team"},
				},
			},
			author:      "alice",
			teamLookup:  teamLookup,
			expectMatch: true,
		},
		{
			name: "user not in team",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Teams: []string{"dev-team"},
				},
			},
			author:      "carol",
			teamLookup:  teamLookup,
			expectMatch: false,
		},
		{
			name: "user matches and in team",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Users: []string{"alice"},
					Teams: []string{"frontend-team"},
				},
			},
			author:      "alice",
			teamLookup:  teamLookup,
			expectMatch: true,
		},
		{
			name: "user matches but not in team",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Users: []string{"alice"},
					Teams: []string{"backend-team"},
				},
			},
			author:      "alice",
			teamLookup:  teamLookup,
			expectMatch: false,
		},
		{
			name: "team check without teamLookup",
			when: &rules.RuleWhen{
				Contributor: &rules.RuleWhenContributor{
					Teams: []string{"dev-team"},
				},
			},
			author:      "alice",
			teamLookup:  nil,
			expectMatch: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &rules.Rule{
				Name: "test-rule",
				When: tt.when,
			}

			prData := &mockPRData{
				author: tt.author,
			}

			match, err := rule.MatchWhen(t.Context(), prData, tt.teamLookup)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectMatch, match)
			}
		})
	}
}

func TestMatchWhen_CombinedConditions(t *testing.T) {
	teamLookup := &mockTeamLookup{
		teams: map[string][]string{
			"alice": {"dev-team"},
		},
	}

	rule := &rules.Rule{
		Name: "test-rule",
		When: &rules.RuleWhen{
			Branch: &rules.RuleWhenBranch{
				Target: []string{"main"},
				Source: []string{"feature/*"},
			},
			File: &rules.RuleWhenFile{
				Include: []string{"src/**"},
			},
			Contributor: &rules.RuleWhenContributor{
				Teams: []string{"dev-team"},
			},
		},
	}

	t.Run("all conditions match", func(t *testing.T) {
		prData := &mockPRData{
			author:       "alice",
			targetBranch: "main",
			sourceBranch: "feature/new",
			files: []rules.PRFile{
				{Filename: "src/main.go"},
			},
		}

		match, err := rule.MatchWhen(t.Context(), prData, teamLookup)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("branch does not match", func(t *testing.T) {
		prData := &mockPRData{
			author:       "alice",
			targetBranch: "develop",
			sourceBranch: "feature/new",
			files: []rules.PRFile{
				{Filename: "src/main.go"},
			},
		}

		match, err := rule.MatchWhen(t.Context(), prData, teamLookup)
		require.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("files do not match", func(t *testing.T) {
		prData := &mockPRData{
			author:       "alice",
			targetBranch: "main",
			sourceBranch: "feature/new",
			files: []rules.PRFile{
				{Filename: "docs/README.md"},
			},
		}

		match, err := rule.MatchWhen(t.Context(), prData, teamLookup)
		require.NoError(t, err)
		assert.False(t, match)
	})
}

func TestMatchFiles(t *testing.T) {
	tests := []struct {
		name        string
		rule        *rules.Rule
		files       []string
		expectMatch bool
		expectError bool
	}{
		{
			name: "no patterns matches all",
			rule: &rules.Rule{
				Files:   []string{},
				Exclude: []string{},
			},
			files:       []string{"src/main.go", "docs/README.md"},
			expectMatch: true,
		},
		{
			name: "files match pattern",
			rule: &rules.Rule{
				Files: []string{"src/**/*.go"},
			},
			files:       []string{"src/main.go", "src/api/handler.go"},
			expectMatch: true,
		},
		{
			name: "files do not match pattern",
			rule: &rules.Rule{
				Files: []string{"src/**/*.go"},
			},
			files:       []string{"docs/README.md"},
			expectMatch: false,
		},
		{
			name: "files match but excluded",
			rule: &rules.Rule{
				Files:   []string{"src/**"},
				Exclude: []string{"src/test/**"},
			},
			files:       []string{"src/test/test.go"},
			expectMatch: false,
		},
		{
			name: "some files match after exclusion",
			rule: &rules.Rule{
				Files:   []string{"**/*.go"},
				Exclude: []string{"**/test/**"},
			},
			files:       []string{"src/main.go", "src/test/test.go"},
			expectMatch: true,
		},
		{
			name: "multiple patterns",
			rule: &rules.Rule{
				Files: []string{"*.md", "*.txt"},
			},
			files:       []string{"README.md", "notes.txt"},
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := tt.rule.MatchFiles(t.Context(), tt.files)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectMatch, match)
			}
		})
	}
}

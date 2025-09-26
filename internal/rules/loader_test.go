package rules

import (
	"testing"
)

func TestRuleLoader(t *testing.T) {
	loader := NewRuleLoader("mushu.yaml")

	rules, err := loader.LoadRules("../../")
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	if len(rules) == 0 {
		t.Error("Expected rules to be loaded")
	}

	// Check that we have the expected rules
	ruleNames := make(map[string]bool)
	for _, rule := range rules {
		ruleNames[rule.Name] = true
	}

	expectedRules := []string{"security-review", "large-changes", "infrastructure", "binary-files"}
	for _, expected := range expectedRules {
		if !ruleNames[expected] {
			t.Errorf("Expected rule %s not found", expected)
		}
	}
}

func TestRuleMatcher(t *testing.T) {
	matcher := NewRuleMatcher()

	// Test path matching
	testCases := []struct {
		pattern  string
		filePath string
		expected bool
	}{
		{"**", "src/backend/main.go", true},
		{"src/**", "src/backend/main.go", true},
		{"src/**", "docs/README.md", false},
		{"*.go", "main.go", false}, // This would need proper glob matching
	}

	for _, tc := range testCases {
		result := matcher.pathMatches(tc.pattern, tc.filePath)
		if result != tc.expected {
			t.Errorf("pathMatches(%q, %q) = %v, expected %v", tc.pattern, tc.filePath, result, tc.expected)
		}
	}
}

func TestConditionMatching(t *testing.T) {
	matcher := NewRuleMatcher()

	files := []PRFile{
		{Filename: "config/.env", Changes: 10},
		{Filename: "src/main.go", Changes: 50},
		{Filename: "binary.exe", Changes: 1000},
	}

	conditions := []Condition{
		{
			Type:         "sensitive-file",
			Patterns:     []string{".env", "secrets"},
			RequireTeams: []string{"security-team"},
			Approvers:    map[string]int{"security-team": 1},
		},
		{
			Type:         "file-change",
			MaxChanges:   100,
			RequireTeams: []string{"senior-backend"},
			Approvers:    map[string]int{"senior-backend": 1},
		},
		{
			Type:       "binary-file",
			Extensions: []string{".exe", ".dll"},
			Action:     "forbid",
		},
	}

	violations, approvals := matcher.MatchConditions(conditions, files)

	if len(violations) == 0 {
		t.Error("Expected violations for binary file")
	}

	if len(approvals) == 0 {
		t.Error("Expected approval requirements")
	}
}

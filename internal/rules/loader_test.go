package rules_test

import (
	"os"
	"testing"

	"github.com/koblas/mushu/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleLoader(t *testing.T) {
	loader := rules.NewRuleLoader(".",
		rules.WithFileSystem(os.DirFS("../../")),
		rules.WithFilename("mushu.yaml"),
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

func TestRuleMatcher(t *testing.T) {
	matcher := rules.NewRuleMatcher()

	rules := []*rules.Rule{
		{Name: "all-files", Files: []string{"**"}},
		{Name: "src-files", Files: []string{"src/**/*.go"}},
		{Name: "docs-files", Files: []string{"docs/**"}},
		{Name: "go-files", Files: []string{"*.go"}},
	}

	// Test path matching
	testCases := []struct {
		filePath string
		expected int
	}{
		{filePath: "src/backend/main.go", expected: 2},
		{filePath: "src/backend/main.go", expected: 2},
		{filePath: "src/README.md", expected: 1},
		{filePath: "docs/README.md", expected: 2},
		{filePath: "main.go", expected: 2}, // This would need proper glob matching
	}

	for _, tc := range testCases {
		result := matcher.MatchRules(t.Context(), rules, []string{tc.filePath})
		var names []string
		for _, r := range result {
			names = append(names, r.Name)
		}
		assert.Len(t, names, tc.expected, "MatchRules(%q) = %d, expected %d", tc.filePath, len(result), tc.expected)
	}
}

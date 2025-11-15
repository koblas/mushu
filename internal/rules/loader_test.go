package rules_test

import (
	"embed"
	_ "embed"
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

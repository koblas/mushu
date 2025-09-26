package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RulesData represents the structure of mushu.yaml files
type RulesData struct {
	Rules []Rule `yaml:"rules"`
}

// Rule represents a single rule definition
type Rule struct {
	Name       string      `yaml:"name"`
	Path       string      `yaml:"path"`
	Inherit    bool        `yaml:"inherit"`
	Conditions []Condition `yaml:"conditions"`
}

// Condition represents a rule condition
type Condition struct {
	Type         string         `yaml:"type"`
	Pattern      string         `yaml:"pattern"`
	Patterns     []string       `yaml:"patterns"`
	MaxChanges   int            `yaml:"max_changes"`
	Extensions   []string       `yaml:"extensions"`
	RequireTeams []string       `yaml:"require_teams"`
	Approvers    map[string]int `yaml:"approvers"`
	Action       string         `yaml:"action"`
}

// RuleLoader handles loading and inheritance of rules
type RuleLoader struct {
	rulesFile string
}

// NewRuleLoader creates a new rule loader
func NewRuleLoader(rulesFile string) *RuleLoader {
	return &RuleLoader{
		rulesFile: rulesFile,
	}
}

// LoadRules loads rules from a directory with inheritance
func (rl *RuleLoader) LoadRules(dir string) ([]Rule, error) {
	var allRules []Rule

	// Walk up the directory tree to find all mushu.yaml files
	currentDir := dir
	for {
		rulesPath := filepath.Join(currentDir, rl.rulesFile)

		if _, err := os.Stat(rulesPath); err == nil {
			rules, err := rl.loadRulesFile(rulesPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load rules from %s: %w", rulesPath, err)
			}

			// Add rules that should be inherited
			for _, rule := range rules {
				if rule.Inherit || currentDir == dir {
					allRules = append(allRules, rule)
				}
			}
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Reached root
		}
		currentDir = parentDir
	}

	return allRules, nil
}

// loadRulesFile loads rules from a single mushu.yaml file
func (rl *RuleLoader) loadRulesFile(filename string) ([]Rule, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var rulesData RulesData
	if err := yaml.Unmarshal(data, &rulesData); err != nil {
		return nil, fmt.Errorf("failed to parse rules file: %w", err)
	}

	return rulesData.Rules, nil
}

// MatchRules returns rules that match the given file path
func MatchRules(rules []Rule, filePath string) []Rule {
	var matched []Rule

	for _, rule := range rules {
		if pathMatches(rule.Path, filePath) {
			matched = append(matched, rule)
		}
	}

	return matched
}

// pathMatches checks if a file path matches a rule path pattern
func pathMatches(pattern, filePath string) bool {
	// Simple glob matching - can be enhanced with proper glob library
	if pattern == "**" {
		return true
	}

	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(filePath, prefix)
	}

	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		dir := filepath.Dir(filePath)
		return strings.HasPrefix(dir, prefix)
	}

	return filePath == pattern
}

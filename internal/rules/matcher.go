package rules

import (
	"path/filepath"
	"strings"
)

// RuleMatcher handles matching rules against file paths
type RuleMatcher struct{}

// NewRuleMatcher creates a new rule matcher
func NewRuleMatcher() *RuleMatcher {
	return &RuleMatcher{}
}

// MatchRules returns rules that match the given file path
func (rm *RuleMatcher) MatchRules(rules []Rule, filePath string) []Rule {
	var matched []Rule

	for _, rule := range rules {
		if rm.pathMatches(rule.Path, filePath) {
			matched = append(matched, rule)
		}
	}

	return matched
}

// pathMatches checks if a file path matches a rule path pattern
func (rm *RuleMatcher) pathMatches(pattern, filePath string) bool {
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

// MatchConditions checks if file changes match rule conditions
func (rm *RuleMatcher) MatchConditions(conditions []Condition, files []PRFile) ([]string, map[string]int) {
	var violations []string
	approvalRequirements := make(map[string]int)

	for _, condition := range conditions {
		switch condition.Type {
		case "sensitive-file":
			if rm.matchesSensitiveFiles(condition.Patterns, files) {
				if len(condition.RequireTeams) > 0 {
					// Add approval requirements
					for team, count := range condition.Approvers {
						approvalRequirements[team] = count
					}
				}
			}

		case "file-change":
			if rm.matchesFileChanges(condition, files) {
				if len(condition.RequireTeams) > 0 {
					// Add approval requirements
					for team, count := range condition.Approvers {
						approvalRequirements[team] = count
					}
				}
			}

		case "binary-file":
			if rm.matchesBinaryFiles(condition.Extensions, files) {
				if condition.Action == "forbid" {
					violations = append(violations, "Binary files are not allowed")
				}
			}
		}
	}

	return violations, approvalRequirements
}

// matchesSensitiveFiles checks if any files match sensitive file patterns
func (rm *RuleMatcher) matchesSensitiveFiles(patterns []string, files []PRFile) bool {
	for _, file := range files {
		for _, pattern := range patterns {
			if strings.Contains(file.Filename, pattern) {
				return true
			}
		}
	}
	return false
}

// matchesFileChanges checks if file changes match the condition
func (rm *RuleMatcher) matchesFileChanges(condition Condition, files []PRFile) bool {
	for _, file := range files {
		// Check pattern match
		if condition.Pattern != "" && !rm.matchesPattern(condition.Pattern, file.Filename) {
			continue
		}

		// Check patterns match
		if len(condition.Patterns) > 0 {
			matched := false
			for _, pattern := range condition.Patterns {
				if rm.matchesPattern(pattern, file.Filename) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Check max changes
		if condition.MaxChanges > 0 && file.Changes > condition.MaxChanges {
			return true
		}
	}

	return false
}

// matchesBinaryFiles checks if any files are binary files
func (rm *RuleMatcher) matchesBinaryFiles(extensions []string, files []PRFile) bool {
	for _, file := range files {
		ext := filepath.Ext(file.Filename)
		for _, binaryExt := range extensions {
			if ext == binaryExt {
				return true
			}
		}
	}
	return false
}

// matchesPattern checks if a filename matches a pattern
func (rm *RuleMatcher) matchesPattern(pattern, filename string) bool {
	if strings.HasSuffix(pattern, "**") {
		prefix := strings.TrimSuffix(pattern, "**")
		return strings.HasPrefix(filename, prefix)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(filename, prefix)
	}

	return filename == pattern
}

// PRFile represents a file in a pull request
type PRFile struct {
	Filename  string
	Status    string // added, modified, removed, renamed
	Additions int
	Deletions int
	Changes   int
}

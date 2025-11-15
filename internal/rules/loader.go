package rules

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
)

// RulesData represents the structure of mushu.yaml files
type RulesData struct {
	Rules []*Rule `yaml:"rules"`
}

// Rule represents a single rule definition
type Rule struct {
	// Unique identifier for the rule
	Id string `yaml:"id"`
	// the name of the hook (plugin) to run
	Name string `yaml:"name"`
	// [optional] description of the hook, shown in the UI
	Description string `yaml:"description"`
	// When condition to run the rule
	When *RuleWhen `yaml:"when"`
	// pattern(s) of files this rule applies to
	Files []string `yaml:"files"`
	// pattern(s) of files to exclude from this rule (inverse of files)
	Exclude []string `yaml:"exclude"`
	// Verbose flag to enable detailed logging (if supported)
	Verbose bool `yaml:"verbose"`
	// Arguments for the rule
	Args map[string]string `yaml:"args"`

	// the source file where this rule was defined
	SourceFile string `yaml:"-"`
	// The source directory where this rule was defined (for scoping)
	SourceDir string `yaml:"-"`
}

type RuleWhen struct {
	Branch      *RuleWhenBranch      `yaml:"branch"`
	File        *RuleWhenFile        `yaml:"file"`
	Contributor *RuleWhenContributor `yaml:"contributor"`
}

type RuleWhenBranch struct {
	Target []string `yaml:"target"`
	Source []string `yaml:"source"`
}

type RuleWhenFile struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type RuleWhenContributor struct {
	Users []string `yaml:"users"`
	Teams []string `yaml:"teams"`
}

// RuleLoader handles loading and inheritance of rules
type RuleLoader struct {
	RuleBase
	// the sub directory to load rules from
	dir string
}

// NewRuleLoader creates a new rule loader
func NewRuleLoader(dir string, opts ...FindOption) *RuleLoader {
	rl := &RuleLoader{
		RuleBase: RuleBase{
			fsys:     os.DirFS("."),
			filename: "mushu.yaml",
		},
		dir: dir,
	}

	for _, opt := range opts {
		opt(&rl.RuleBase)
	}

	return rl
}

// LoadRules loads rules from a directory with inheritance
func (rl *RuleLoader) LoadRules(ctx context.Context) ([]*Rule, error) {
	var allRules []*Rule

	// Walk up the directory tree to find all mushu.yaml files
	for dir := range rl.walkIterator(rl.dir) {
		contents, err := rl.loadRules(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// No rules found in this path, continue
				continue
			}
		}

		allRules = append(allRules, contents...)
	}

	return allRules, nil
}

// MatchWhen evaluates the when condition against PR data
func (r *Rule) MatchWhen(ctx context.Context, prData PRData, teamLookup TeamLookup) (bool, error) {
	// If no when condition is specified, the rule matches
	if r.When == nil {
		return true, nil
	}

	// Check branch conditions
	if r.When.Branch != nil {
		if !matchBranch(r.When.Branch, prData) {
			return false, nil
		}
	}

	// Check file conditions
	if r.When.File != nil {
		files := prData.GetFiles()
		fileNames := make([]string, len(files))
		for i, f := range files {
			fileNames[i] = f.Filename
		}
		match, err := matchFiles(r.When.File, fileNames)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}

	// Check contributor conditions
	if r.When.Contributor != nil {
		match, err := matchContributor(ctx, r.When.Contributor, prData, teamLookup)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}

	return true, nil
}

// matchBranch checks if branch conditions are met
func matchBranch(branch *RuleWhenBranch, prData PRData) bool {
	// Check target branch
	if len(branch.Target) > 0 {
		if !matchPattern(prData.GetTargetBranch(), branch.Target) {
			return false
		}
	}

	// Check source branch
	if len(branch.Source) > 0 {
		if !matchPattern(prData.GetSourceBranch(), branch.Source) {
			return false
		}
	}

	return true
}

// matchFiles checks if file conditions are met
func matchFiles(fileCondition *RuleWhenFile, files []string) (bool, error) {
	matches := make(map[string]struct{})

	// If include patterns are specified, match against them
	if len(fileCondition.Include) > 0 {
		for _, pattern := range fileCondition.Include {
			for _, filePath := range files {
				match, err := doublestar.Match(pattern, filePath)
				if err != nil {
					return false, fmt.Errorf("error matching include pattern %q: %w", pattern, err)
				}
				if match {
					matches[filePath] = struct{}{}
				}
			}
		}
	} else {
		// If no include patterns, all files match by default
		for _, filePath := range files {
			matches[filePath] = struct{}{}
		}
	}

	// Apply exclude patterns
	for _, pattern := range fileCondition.Exclude {
		for filePath := range matches {
			match, err := doublestar.Match(pattern, filePath)
			if err != nil {
				return false, fmt.Errorf("error matching exclude pattern %q: %w", pattern, err)
			}
			if match {
				delete(matches, filePath)
			}
		}
	}

	return len(matches) > 0, nil
}

// matchContributor checks if contributor conditions are met
func matchContributor(ctx context.Context, contributor *RuleWhenContributor, prData PRData, teamLookup TeamLookup) (bool, error) {
	// Check if author is in the users list
	if len(contributor.Users) > 0 {
		if !slices.Contains(contributor.Users, prData.GetAuthor()) {
			return false, nil
		}
	}

	// Check if author belongs to any of the specified teams
	if len(contributor.Teams) > 0 {
		if teamLookup == nil {
			return false, fmt.Errorf("team lookup required but not provided")
		}

		authorTeams, err := teamLookup.GetUserTeams(ctx, prData.GetAuthor())
		if err != nil {
			return false, fmt.Errorf("failed to get user teams: %w", err)
		}

		found := false
		for _, team := range contributor.Teams {
			if slices.Contains(authorTeams, team) {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}

// matchPattern checks if a value matches any pattern (supports wildcards)
func matchPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		match, err := doublestar.Match(pattern, value)
		if err == nil && match {
			return true
		}
	}
	return false
}

func (r *Rule) MatchFiles(ctx context.Context, files []string) (bool, error) {
	matches := make(map[string]struct{}, len(files))

	if len(r.Files) == 0 {
		for _, filePath := range files {
			matches[filePath] = struct{}{}
		}
	} else {
		for _, pattern := range r.Files {
			for _, filePath := range files {
				if match, err := doublestar.Match(pattern, filePath); err != nil {
					return false, fmt.Errorf("error matching pattern %q: %w", pattern, err)
				} else if match {
					matches[filePath] = struct{}{}
				}
			}
		}
	}

	for _, pattern := range r.Exclude {
		for _, filePath := range files {
			if match, err := doublestar.Match(pattern, filePath); err != nil {
				return false, fmt.Errorf("error matching exclude pattern %q: %w", pattern, err)
			} else if match {
				delete(matches, filePath)
			}
		}
	}

	return len(matches) > 0, nil
}

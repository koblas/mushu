package rules

import (
	"context"
	"errors"
	"fmt"
	"os"

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
			} else if !match {
				delete(matches, filePath)
			}
		}
	}

	return len(matches) > 0, nil
}

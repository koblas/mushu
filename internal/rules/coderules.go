package rules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
)

// CodeRulesFinder handles discovery of CODERULES files
type CodeRulesFinder struct {
	RuleBase

	prFiles []PRFile
}

// NewCodeRulesFinder creates a new CODERULES finder
// If filename is empty, defaults to "CODERULES"
func NewPrRuleLoader(prFiles []PRFile, opts ...FindOption) *CodeRulesFinder {
	crf := &CodeRulesFinder{
		RuleBase: RuleBase{
			fsys:     os.DirFS("."),
			filename: "mushu.yaml",
		},
		prFiles: prFiles,
	}

	for _, opt := range opts {
		opt(&crf.RuleBase)
	}

	return crf
}

// FindCodeRulesForPR discovers all CODERULES files in directories affected by PR files
// Returns a deduplicated list of CODERULES files, ordered from deepest to shallowest
func (crf *CodeRulesFinder) LoadRules(ctx context.Context) ([]*Rule, error) {
	// Extract unique directories from PR files
	directories := crf.GetAffectedDirectories(crf.prFiles)

	// Find CODERULES files in those directories and parent directories
	var rules []*Rule

	for _, dir := range directories {
		contents, err := crf.loadRules(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// No CODERULES found in this path, continue
				continue
			}

			return nil, fmt.Errorf("unable to open rules in %q: %w", dir, err)
		}

		rules = append(rules, contents...)
	}

	return rules, nil
}

// GetAffectedDirectories returns a list of unique directories affected by the PR files
func (crf *CodeRulesFinder) GetAffectedDirectories(prFiles []PRFile) []string {
	dirMap := map[string]struct{}{
		".": {},
	}

	for _, file := range prFiles {
		for dir := path.Dir(file.Filename); dir != "."; dir = path.Join(dir, "..") {
			dirMap[dir] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(dirMap))
}

func (crf *CodeRulesFinder) RulesForPath(filename string, rules []*Rule) ([]*Rule, error) {
	var result []*Rule

	byDir := map[string][]*Rule{}
	for _, r := range rules {
		byDir[r.SourceDir] = append(byDir[r.SourceDir], r)
	}

	for dir := range crf.walkIterator(path.Dir(filename)) {
		if r, ok := byDir[dir]; ok {
			result = append(result, r...)
		}
	}

	return result, nil
}

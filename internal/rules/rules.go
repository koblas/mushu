package rules

import (
	"fmt"
	"io"
	"io/fs"
	"iter"
	"path"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type RuleBase struct {
	fsys     fs.FS
	filename string // Name of the rules file to search for (e.g., "CODERULES")
}

// CodeRulesFile represents a CODERULES file and its location
type CodeRulesFile struct {
	Path      string  // Full path to the CODERULES file
	Directory string  // Directory containing the CODERULES file
	Rules     []*Rule // Parsed rules from the file, in order of most specific to least specific
}

type FindOption func(*RuleBase)

func WithFileSystem(fsys fs.FS) FindOption {
	return func(rl *RuleBase) {
		rl.fsys = fsys
	}
}

func WithFilename(name string) FindOption {
	return func(rl *RuleBase) {
		rl.filename = name
	}
}

func (crf *RuleBase) walkIterator(base string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for dir := base; dir != "."; dir = path.Join(dir, "..") {
			if !yield(dir) {
				return
			}
		}

		yield(".")
	}
}

func (crf *RuleBase) loadRules(dir string) ([]*Rule, error) {
	rulesPath := filepath.Join(dir, crf.filename)
	fd, err := crf.fsys.Open(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rules %q in %q: %w", dir, crf.filename, err)
	}

	defer fd.Close()

	data, err := io.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules %q: %w", rulesPath, err)
	}

	var rulesData RulesData
	if err := yaml.Unmarshal(data, &rulesData); err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", rulesPath, err)
	}

	for _, rule := range rulesData.Rules {
		rule.SourceFile = rulesPath
		rule.SourceDir = dir
	}

	return rulesData.Rules, nil
}

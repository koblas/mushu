package rules_test

import (
	"testing"
	"testing/fstest"

	"github.com/koblas/mushu/internal/rules"
	"github.com/stretchr/testify/assert"
)

func TestFindCodeRulesForPR(t *testing.T) {
	// Create a mock filesystem with CODERULES files
	fsys := fstest.MapFS{
		"CODERULES": &fstest.MapFile{
			Data: []byte(`rules:
  - name: root-rule
    path: "**"
    conditions:
      - type: sensitive-file
        patterns: [".env"]
`),
		},
		"src/CODERULES": &fstest.MapFile{
			Data: []byte(`rules:
  - name: src-rule
    path: "src/**"
    conditions:
      - type: file-change
        max_changes: 100
`),
		},
		"src/api/CODERULES": &fstest.MapFile{
			Data: []byte(`rules:
  - name: api-rule
    path: "src/api/**"
    conditions:
      - type: binary-file
        extensions: [".so", ".dylib"]
        action: forbid
`),
		},
	}

	tests := []struct {
		name          string
		prFiles       []rules.PRFile
		expectedCount int
		expectedPaths []string
	}{
		{
			name: "files in api directory",
			prFiles: []rules.PRFile{
				{Filename: "src/api/handler.go"},
				{Filename: "src/api/types.go"},
			},
			expectedCount: 3,
			expectedPaths: []string{"src/api/CODERULES", "src/CODERULES", "CODERULES"},
		},
		{
			name: "files in src directory",
			prFiles: []rules.PRFile{
				{Filename: "src/main.go"},
			},
			expectedCount: 2,
			expectedPaths: []string{"src/CODERULES", "CODERULES"},
		},
		{
			name: "files in root",
			prFiles: []rules.PRFile{
				{Filename: "README.md"},
			},
			expectedCount: 1,
			expectedPaths: []string{"CODERULES"},
		},
		{
			name: "files in multiple directories",
			prFiles: []rules.PRFile{
				{Filename: "src/api/handler.go"},
			},
			expectedCount: 3, // Deduplicated
			expectedPaths: []string{"src/api/CODERULES", "src/CODERULES", "CODERULES"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := rules.NewPrRuleLoader(tt.prFiles,
				rules.WithFileSystem(fsys),
				rules.WithFilename("CODERULES"),
			)

			loaded, err := finder.LoadRules(t.Context())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assert.Len(t, loaded, tt.expectedCount, "number of CODERULES files")

			rules, err := finder.RulesForPath(tt.prFiles[0].Filename, loaded)
			assert.NoError(t, err)

			var paths []string
			for _, cr := range rules {
				paths = append(paths, cr.SourceFile)
			}

			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestExtractDirectories(t *testing.T) {
	tests := []struct {
		name     string
		prFiles  []rules.PRFile
		expected []string
	}{
		{
			name: "simple files",
			prFiles: []rules.PRFile{
				{Filename: "src/main.go"},
				{Filename: "src/api/handler.go"},
			},
			expected: []string{"src/api", "src", "."},
		},
		{
			name: "root file",
			prFiles: []rules.PRFile{
				{Filename: "README.md"},
			},
			expected: []string{"."},
		},
		{
			name: "deduplicate directories",
			prFiles: []rules.PRFile{
				{Filename: "src/a.go"},
				{Filename: "src/b.go"},
				{Filename: "src/c.go"},
			},
			expected: []string{"src", "."},
		},
		{
			name: "nested paths",
			prFiles: []rules.PRFile{
				{Filename: "a/b/c/d/file.go"},
			},
			expected: []string{"a/b/c/d", "a/b/c", "a/b", "a", "."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := rules.NewPrRuleLoader(tt.prFiles)
			result := finder.GetAffectedDirectories(tt.prFiles)

			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

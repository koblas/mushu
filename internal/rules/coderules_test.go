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
			finder := rules.NewPrRuleLoader(tt.prFiles, rules.WithFileSystem(fsys))

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

// func TestParseCodeRulesFile(t *testing.T) {
// 	finder := rules.NewCodeRulesFinder()

// 	validRules := []byte(`rules:
//   - name: test-rule
//     path: "src/**"
//     conditions:
//       - type: sensitive-file
//         patterns: [".env", ".key"]
// `)

// 	rules, err := finder.parseCodeRulesFile(validRules)
// 	if err != nil {
// 		t.Fatalf("failed to parse valid rules: %v", err)
// 	}

// 	if len(rules) != 1 {
// 		t.Errorf("expected 1 rule, got %d", len(rules))
// 	}

// 	if rules[0].Name != "test-rule" {
// 		t.Errorf("expected rule name 'test-rule', got '%s'", rules[0].Name)
// 	}

// 	if rules[0].Path != "src/**" {
// 		t.Errorf("expected rule path 'src/**', got '%s'", rules[0].Path)
// 	}

// 	if len(rules[0].Conditions) != 1 {
// 		t.Errorf("expected 1 condition, got %d", len(rules[0].Conditions))
// 	}
// }

// func TestParseCodeRulesFile_Invalid(t *testing.T) {
// 	finder := NewCodeRulesFinder("")

// 	invalidRules := []byte(`invalid yaml [content`)

// 	_, err := finder.parseCodeRulesFile(invalidRules)
// 	if err == nil {
// 		t.Error("expected error parsing invalid YAML, got nil")
// 	}
// }

// func TestFindCodeRulesInPath(t *testing.T) {
// 	fsys := fstest.MapFS{
// 		"CODERULES": &fstest.MapFile{
// 			Data: []byte(`rules:
//   - name: root-rule
//     path: "**"
// `),
// 		},
// 		"src/CODERULES": &fstest.MapFile{
// 			Data: []byte(`rules:
//   - name: src-rule
//     path: "src/**"
// `),
// 		},
// 		"src/api/CODERULES": &fstest.MapFile{
// 			Data: []byte(`rules:
//   - name: api-rule
//     path: "src/api/**"
// `),
// 		},
// 	}

// 	finder := rules.NewCodeRulesFinder(rules.WithFileSystem(fsys))

// 	tests := []struct {
// 		name          string
// 		dir           string
// 		expectedCount int
// 		expectedPaths []string
// 	}{
// 		{
// 			name:          "api directory",
// 			dir:           "src/api",
// 			expectedCount: 3,
// 			expectedPaths: []string{"src/api/CODERULES", "src/CODERULES", "CODERULES"},
// 		},
// 		{
// 			name:          "src directory",
// 			dir:           "src",
// 			expectedCount: 2,
// 			expectedPaths: []string{"src/CODERULES", "CODERULES"},
// 		},
// 		{
// 			name:          "root directory",
// 			dir:           "",
// 			expectedCount: 1,
// 			expectedPaths: []string{"CODERULES"},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			result, err := finder.findCodeRulesInPath(tt.dir)
// 			if err != nil {
// 				t.Fatalf("unexpected error: %v", err)
// 			}

// 			if len(result) != tt.expectedCount {
// 				t.Errorf("expected %d CODERULES files, got %d", tt.expectedCount, len(result))
// 			}

// 			for i, expected := range tt.expectedPaths {
// 				if i >= len(result) {
// 					break
// 				}
// 				if result[i].Path != expected {
// 					t.Errorf("expected path %s at index %d, got %s", expected, i, result[i].Path)
// 				}
// 			}
// 		})
// 	}
// }

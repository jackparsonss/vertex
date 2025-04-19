package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestExtractModuleName(t *testing.T) {
	vp := VertexParser{}
	tempDir, err := os.MkdirTemp("", "go-mod-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testCases := []struct {
		name           string
		fileContent    string
		expectedModule string
		expectError    bool
	}{
		{
			name:           "basic module",
			fileContent:    "module github.com/example/project\n\ngo 1.20\n",
			expectedModule: "github.com/example/project",
			expectError:    false,
		},
		{
			name:           "module with version",
			fileContent:    "module github.com/example/project/v2\n\ngo 1.20\n",
			expectedModule: "github.com/example/project/v2",
			expectError:    false,
		},
		{
			name:           "module with indentation",
			fileContent:    "\t module  github.com/example/project \n\ngo 1.20\n",
			expectedModule: "github.com/example/project",
			expectError:    false,
		},
		{
			name:           "empty file",
			fileContent:    "",
			expectedModule: "",
			expectError:    true,
		},
		{
			name:           "no module declaration",
			fileContent:    "go 1.20\n\nrequire (\n\tgithub.com/pkg/errors v0.9.1\n)\n",
			expectedModule: "",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "go.mod")
			err := os.WriteFile(filePath, []byte(tc.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			moduleName, err := vp.ParseGoMod(filePath)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if moduleName != tc.expectedModule {
				t.Errorf("Expected module name '%s', got '%s'", tc.expectedModule, moduleName)
			}
		})
	}
}

func TestExtractModuleNameFileNotExist(t *testing.T) {
	vp := VertexParser{}
	_, err := vp.ParseGoMod("this-file-does-not-exist.mod")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}
}

func TestAddVertexReplace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go-mod-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testCases := []struct {
		name            string
		initialContent  string
		shouldHaveAdded bool
	}{
		{
			name:            "empty file",
			initialContent:  "",
			shouldHaveAdded: true,
		},
		{
			name:            "file without replace",
			initialContent:  "module example.com/myproject\n\ngo 1.20\n",
			shouldHaveAdded: true,
		},
		{
			name:            "file with different replace",
			initialContent:  "module example.com/myproject\n\ngo 1.20\n\nreplace example.com/other => ./other\n",
			shouldHaveAdded: true,
		},
		{
			name:            "file with exact replace",
			initialContent:  "module example.com/myproject\n\ngo 1.20\n\nreplace vertex => ./vertex\n",
			shouldHaveAdded: false,
		},
		{
			name:            "file with replace with spaces",
			initialContent:  "module example.com/myproject\n\ngo 1.20\n\nreplace   vertex   =>   ./vertex\n",
			shouldHaveAdded: false,
		},
		{
			name:            "file with replace in replace block",
			initialContent:  "module example.com/myproject\n\ngo 1.20\n\nreplace (\n\tvertex => ./vertex\n)\n",
			shouldHaveAdded: false,
		},
	}

	vp := VertexParser{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "go.mod")
			err := os.WriteFile(filePath, []byte(tc.initialContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			err = vp.AddGoModReplace(filePath)
			if err != nil {
				t.Fatalf("Function returned error: %v", err)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file after execution: %v", err)
			}

			replacePattern := regexp.MustCompile(`replace\s+vertex\s*=>\s*\./vertex`)
			hasReplace := replacePattern.MatchString(string(content))

			if !hasReplace {
				t.Error("Replace directive was not found in the file after execution")
			}

			if !tc.shouldHaveAdded {
				// Replace directives already existed, so the file should be unchanged
				if !strings.Contains(string(content), tc.initialContent) {
					t.Error("File was modified when it shouldn't have been")
				}
			}

			if tc.shouldHaveAdded {
				if !strings.Contains(string(content), "\nreplace vertex => ./vertex\n") {
					t.Error("Replace directive was not added with the correct formatting")
				}
			}
		})
	}
}

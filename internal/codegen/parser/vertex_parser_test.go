package parser

import (
	"os"
	"path/filepath"
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

package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/jackparsonss/vertex/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseComment(t *testing.T) {
	vp := &VertexParser{}

	tests := []struct {
		name           string
		commentCode    string
		expectedPath   string
		expectedMethod string
	}{
		{
			name:           "Both path and method in same comment",
			commentCode:    "// @server path=/api/users method=GET\nfunc GetUsers() {}",
			expectedPath:   "/api/users",
			expectedMethod: "GET",
		},
		{
			name:           "No server directive",
			commentCode:    "// This is just a regular comment\nfunc RegularFunction() {}",
			expectedPath:   "",
			expectedMethod: "",
		},
		{
			name: "Multiple server directives",
			commentCode: `// @server path=/api/v1/users method=GET
			// @server path=/api/v2/users method=POST
			func UserEndpoint() {}`,
			expectedPath:   "/api/v2/users",
			expectedMethod: "POST",
		},
		{
			name:           "Path at end of comment",
			commentCode:    "// @server method=PUT path=/api/update\nfunc UpdateResource() {}",
			expectedPath:   "/api/update",
			expectedMethod: "PUT",
		},
		{
			name:           "No doc comment",
			commentCode:    "func NoComment() {}",
			expectedPath:   "",
			expectedMethod: "",
		},
		{
			name:           "Complex path with parameters",
			commentCode:    "// @server path=/api/users/{id}/posts?sort=desc method=GET\nfunc GetUserPosts() {}",
			expectedPath:   "/api/users/{id}/posts?sort=desc",
			expectedMethod: "GET",
		},
		{
			name:           "Whitespace in directives",
			commentCode:    "// @server path = /api/resources method = PATCH\nfunc PatchResource() {}",
			expectedPath:   "/api/resources",
			expectedMethod: "PATCH",
		},
		{
			name:           "Empty directives",
			commentCode:    "// @server path= method=\nfunc EmptyDirectives() {}",
			expectedPath:   "",
			expectedMethod: "",
		},
		{
			name:           "Multiple spaces between directives",
			commentCode:    "// @server path=/api/v1      method=GET\nfunc WithSpaces() {}",
			expectedPath:   "/api/v1",
			expectedMethod: "GET",
		},
		{
			name:           "Path with special characters",
			commentCode:    "// @server path=/api/v1/data-export/{format} method=GET\nfunc ExportData() {}",
			expectedPath:   "/api/v1/data-export/{format}",
			expectedMethod: "GET",
		},
		{
			name:           "Method with lowercase",
			commentCode:    "// @server path=/api/data method=get\nfunc GetDataLowercase() {}",
			expectedPath:   "/api/data",
			expectedMethod: "get",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn := parseFunctionCode(t, tc.commentCode)

			path, method := vp.parseComment(fn)

			assert.Equal(t, tc.expectedPath, path, "Path should match expected value")
			assert.Equal(t, tc.expectedMethod, method, "Method should match expected value")
		})
	}
}

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

func TestRunGoModTidy(t *testing.T) {
	vp := VertexParser{}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("skipping test; 'go' command not available")
	}

	tempDir, err := os.MkdirTemp("", "go-mod-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := `module example.com/validmodule

go 1.24
`
	err = os.WriteFile(goModPath, []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err = vp.RunGoModTidy(goModPath)
	if err != nil {
		t.Errorf("runGoModTidy failed on valid module: %v", err)
	}
}

func TestParseReturnType(t *testing.T) {
	tests := []struct {
		name            string
		functionCode    string
		structsMap      types.DeclarationMap
		expectedType    string
		expectedIsSlice bool
	}{
		{
			name: "No return type",
			functionCode: `
				func NoReturn() {
					// Function with no return type
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "",
			expectedIsSlice: false,
		},
		{
			name: "Basic return type",
			functionCode: `
				func BasicReturn() string {
					return "hello"
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "string",
			expectedIsSlice: false,
		},
		{
			name: "Pointer return type",
			functionCode: `
				func PointerReturn() *int {
					x := 42
					return &x
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "*int",
			expectedIsSlice: false,
		},
		{
			name: "Slice return type",
			functionCode: `
				func SliceReturn() []string {
					return []string{"hello", "world"}
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "[]string",
			expectedIsSlice: true,
		},
		{
			name: "Map return type",
			functionCode: `
				func MapReturn() map[string]int {
					return map[string]int{"one": 1}
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "map[string]int",
			expectedIsSlice: false,
		},
		{
			name: "Custom type return",
			functionCode: `
				func CustomReturn() CustomType {
					return CustomType{}
				}
			`,
			structsMap:      types.DeclarationMap{"CustomType": "mypackage"},
			expectedType:    "mypackage.CustomType",
			expectedIsSlice: false,
		},
		{
			name: "Slice of custom type return",
			functionCode: `
				func CustomSliceReturn() []CustomType {
					return []CustomType{}
				}
			`,
			structsMap:      types.DeclarationMap{"CustomType": "mypackage"},
			expectedType:    "[]mypackage.CustomType",
			expectedIsSlice: true,
		},
		{
			name: "Multiple return values (should take first only)",
			functionCode: `
				func MultipleReturn() (string, error) {
					return "result", nil
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "string",
			expectedIsSlice: false,
		},
		{
			name: "Selector expression return type",
			functionCode: `
				func SelectorReturn() fmt.Stringer {
					return nil
				}
			`,
			structsMap:      types.DeclarationMap{},
			expectedType:    "fmt.Stringer",
			expectedIsSlice: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcDecl := parseFunctionCode(t, tt.functionCode)

			vp := NewVertexParser([]*ast.File{}, config.Config{})

			returnType, isSlice := vp.parseReturnType(funcDecl, tt.structsMap)

			assert.Equal(t, tt.expectedType, returnType, "Return type should match expected")
			assert.Equal(t, tt.expectedIsSlice, isSlice, "IsSlice flag should match expected")
		})
	}
}

func parseFunctionCode(t *testing.T, code string) *ast.FuncDecl {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", "package main\n"+code, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			return funcDecl
		}
	}

	t.Fatalf("Could not find function declaration in code")
	return nil
}

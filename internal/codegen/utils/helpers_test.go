package utils

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/stretchr/testify/assert"
)

func TestGetTypeString(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		typeMap  types.DeclarationMap
		expected string
	}{
		{
			name:     "Identifier",
			code:     "var x int",
			typeMap:  types.DeclarationMap{},
			expected: "int",
		},
		{
			name:     "Default case - FuncType",
			code:     "var x func(int) string",
			typeMap:  types.DeclarationMap{},
			expected: "func(int) string",
		},
		{
			name:     "Default case - FuncType",
			code:     "var x func(int, string) (string, bool)",
			typeMap:  types.DeclarationMap{},
			expected: "func(int, string) (string, bool)",
		},
		{
			name:     "Default case - FuncType",
			code:     "var x func(...string)",
			typeMap:  types.DeclarationMap{},
			expected: "func(...string)",
		},
		{
			name:     "Default case - channel",
			code:     "var x func(ch chan float)",
			typeMap:  types.DeclarationMap{},
			expected: "func(chan float)",
		},
		{
			name:     "Default case - in channel",
			code:     "var x func(ch <-chan float)",
			typeMap:  types.DeclarationMap{},
			expected: "func(<-chan float)",
		},
		{
			name:     "Default case - out channel",
			code:     "var x func(ch chan<- float)",
			typeMap:  types.DeclarationMap{},
			expected: "func(chan<- float)",
		},
		{
			name:     "Identifier with package prefix",
			code:     "var x CustomType",
			typeMap:  types.DeclarationMap{"CustomType": "mypackage"},
			expected: "mypackage.CustomType",
		},
		{
			name:     "Pointer type",
			code:     "var x *string",
			typeMap:  types.DeclarationMap{},
			expected: "*string",
		},
		{
			name:     "Pointer to custom type with package",
			code:     "var x *CustomType",
			typeMap:  types.DeclarationMap{"CustomType": "mypackage"},
			expected: "*mypackage.CustomType",
		},
		{
			name:     "Selector Expression",
			code:     "var x fmt.Stringer",
			typeMap:  types.DeclarationMap{},
			expected: "fmt.Stringer",
		},
		{
			name:     "Array type",
			code:     "var x []int",
			typeMap:  types.DeclarationMap{},
			expected: "[]int",
		},
		{
			name:     "Array type",
			code:     "var x [3]int",
			typeMap:  types.DeclarationMap{},
			expected: "[3]int",
		},
		{
			name:     "Array of custom type with package",
			code:     "var x []CustomType",
			typeMap:  types.DeclarationMap{"CustomType": "mypackage"},
			expected: "[]mypackage.CustomType",
		},
		{
			name:     "Map type",
			code:     "var x map[string]int",
			typeMap:  types.DeclarationMap{},
			expected: "map[string]int",
		},
		{
			name:     "Map with custom types and packages",
			code:     "var x map[KeyType]ValueType",
			typeMap:  types.DeclarationMap{"KeyType": "pkg1", "ValueType": "pkg2"},
			expected: "map[pkg1.KeyType]pkg2.ValueType",
		},
		{
			name:     "Struct type",
			code:     "var x struct{}",
			typeMap:  types.DeclarationMap{},
			expected: "struct{}",
		},
		{
			name:     "Complex Struct type",
			code:     "var x struct{Name string\nFoo int}",
			typeMap:  types.DeclarationMap{},
			expected: "struct{Name string\nFoo int}",
		},
		{
			name:     "Interface type",
			code:     "var x interface{}",
			typeMap:  types.DeclarationMap{},
			expected: "interface{}",
		},
		{
			name:     "Complex Interface type",
			code:     "var x interface{Name() string\nFoo() int}",
			typeMap:  types.DeclarationMap{},
			expected: "interface{Name() string\nFoo() int}",
		},
		{
			name:     "Complex Type - Pointer to Array of Maps",
			code:     "var x *[]map[string]int",
			typeMap:  types.DeclarationMap{},
			expected: "*[]map[string]int",
		},
		{
			name:     "Complex Type with custom types",
			code:     "var x *[]map[KeyType]ValueType",
			typeMap:  types.DeclarationMap{"KeyType": "pkg1", "ValueType": "pkg2"},
			expected: "*[]map[pkg1.KeyType]pkg2.ValueType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseTypeExpr(t, tt.code)
			result := GetTypeString(expr, tt.typeMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTypeString_DefaultCase(t *testing.T) {
	expr := &ast.BasicLit{
		Kind:  token.STRING,
		Value: `"hello"`,
	}
	typeMap := types.DeclarationMap{}

	result := GetTypeString(expr, typeMap)

	assert.Equal(t, "*ast.BasicLit", result)
}

func parseTypeExpr(t *testing.T, code string) ast.Expr {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", "package main\n"+code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if valSpec, ok := spec.(*ast.ValueSpec); ok && valSpec.Type != nil {
					return valSpec.Type
				}
			}
		}
	}

	t.Fatalf("Could not find type expression in code: %s", code)
	return nil
}

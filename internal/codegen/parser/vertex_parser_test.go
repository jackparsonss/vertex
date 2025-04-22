package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/jackparsonss/vertex/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go-mod-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "go.mod")
	err = os.WriteFile(filePath, []byte("module vertex_test"), 0644)
	assert.NoError(t, err)

	file1 := parseGoFile(t, `package vertex_pkg_two
		// @server path=/data method=POST
		func SaveData(data string) error { return nil }
	`)
	file2 := parseGoFile(t, `package vertex_pkg_one
		// @server path=/items method=GET
		func GetData() (string, error) { return nil }
	`)
	vp := NewVertexParser([]*ast.File{file1, file2}, config.Config{InputDir: tempDir, GoModFile: filePath})

	v, err := vp.Parse()
	assert.NoError(t, err)

	assert.Equal(t, "vertex_test", v.GoModPackage)
	assert.Len(t, v.Functions, 2)
	assert.Equal(t, types.FunctionInfo{
		Name:             "SaveData",
		Path:             "/data",
		Method:           "POST",
		ReturnType:       "error",
		IsSlice:          false,
		IsMethod:         false,
		ReceiverTypeName: "",
		StructName:       "",
		PackageName:      "vertex_pkg_two",
		Params:           []types.ParamInfo{{Name: "data", Type: "string"}},
	}, v.Functions[0])
	assert.Equal(t, types.FunctionInfo{
		Name:             "GetData",
		Path:             "/items",
		Method:           "GET",
		ReturnType:       "string",
		IsSlice:          false,
		IsMethod:         false,
		ReceiverTypeName: "",
		StructName:       "",
		PackageName:      "vertex_pkg_one",
		Params:           nil,
	}, v.Functions[1])

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file after execution: %v", err)
	}

	replacePattern := regexp.MustCompile(`replace\s+vertex\s*=>\s*\./vertex`)
	hasReplace := replacePattern.MatchString(string(content))

	if !hasReplace {
		t.Error("Replace directive was not found in the file after execution")
	}
}

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
			name:           "No path directive",
			commentCode:    "// @server method=GET\nfunc RegularFunction() {}",
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

func TestParseParams(t *testing.T) {
	vp := VertexParser{}

	tests := []struct {
		name           string
		functionCode   string
		expectedParams []types.ParamInfo
		structsMap     types.DeclarationMap
	}{
		{
			name:           "No parameters",
			functionCode:   "func NoParams() {}",
			expectedParams: []types.ParamInfo{},
			structsMap:     types.DeclarationMap{},
		},
		{
			name:         "Single named parameter",
			functionCode: "func SingleParam(name string) {}",
			expectedParams: []types.ParamInfo{
				{Name: "name", Type: "string"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Multiple parameters of different types",
			functionCode: "func MultipleParams(id int, name string, active bool) {}",
			expectedParams: []types.ParamInfo{
				{Name: "id", Type: "int"},
				{Name: "name", Type: "string"},
				{Name: "active", Type: "bool"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Multiple parameters of same type",
			functionCode: "func SameTypeParams(first, second string) {}",
			expectedParams: []types.ParamInfo{
				{Name: "first", Type: "string"},
				{Name: "second", Type: "string"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Unnamed parameter",
			functionCode: "func UnnamedParam(string) {}",
			expectedParams: []types.ParamInfo{
				{Name: "param0", Type: "string"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Array parameter",
			functionCode: "func ArrayParam(items []string) {}",
			expectedParams: []types.ParamInfo{
				{Name: "items", Type: "[]string"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Map parameter",
			functionCode: "func MapParam(data map[string]interface{}) {}",
			expectedParams: []types.ParamInfo{
				{Name: "data", Type: "map[string]interface{}"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Pointer parameter",
			functionCode: "func PointerParam(user *User) {}",
			expectedParams: []types.ParamInfo{
				{Name: "user", Type: "*User"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Channel parameter",
			functionCode: "func ChannelParam(ch chan string) {}",
			expectedParams: []types.ParamInfo{
				{Name: "ch", Type: "chan string"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Function parameter",
			functionCode: "func FuncParam(callback func(int) bool) {}",
			expectedParams: []types.ParamInfo{
				{Name: "callback", Type: "func(int) bool"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Interface parameter",
			functionCode: "func InterfaceParam(data interface{}) {}",
			expectedParams: []types.ParamInfo{
				{Name: "data", Type: "interface{}"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Mixed complex parameters",
			functionCode: "func ComplexParams(id int, users []*User, options map[string]interface{}) {}",
			expectedParams: []types.ParamInfo{
				{Name: "id", Type: "int"},
				{Name: "users", Type: "[]*User"},
				{Name: "options", Type: "map[string]interface{}"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Variadic parameter",
			functionCode: "func VariadicParam(messages ...string) {}",
			expectedParams: []types.ParamInfo{
				{Name: "messages", Type: "...string"},
			},
			structsMap: types.DeclarationMap{},
		},
		{
			name:         "Package qualified types",
			functionCode: "func QualifiedTypes(t time.Time, b bytes.Buffer) {}",
			expectedParams: []types.ParamInfo{
				{Name: "t", Type: "time.Time"},
				{Name: "b", Type: "bytes.Buffer"},
			},
			structsMap: types.DeclarationMap{
				"Time":   "time",
				"Buffer": "bytes",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn := parseFunctionCode(t, tc.functionCode)
			params := vp.parseParams(fn, tc.structsMap)

			assert.Equal(t, len(tc.expectedParams), len(params),
				"Number of parameters should match")

			for i, expected := range tc.expectedParams {
				if i < len(params) {
					assert.Equal(t, expected.Name, params[i].Name,
						"Parameter name at index %d should match", i)
					assert.Equal(t, expected.Type, params[i].Type,
						"Parameter type at index %d should match", i)
				}
			}
		})
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

func TestParseStructDeclarations(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectedMap types.DeclarationMap
	}{
		{
			name:        "No declarations",
			code:        `package testpkg`,
			expectedMap: types.DeclarationMap{},
		},
		{
			name: "Single struct declaration",
			code: `
				package testpkg
				type MyStruct struct {}
			`,
			expectedMap: types.DeclarationMap{"MyStruct": "testpkg"},
		},
		{
			name: "Multiple struct declarations",
			code: `
				package testpkg
				type StructA struct {}
				type StructB struct {}
			`,
			expectedMap: types.DeclarationMap{
				"StructA": "testpkg",
				"StructB": "testpkg",
			},
		},
		{
			name: "Mixed type declarations",
			code: `
				package testpkg
				type StructA struct {}
				type AliasA = int
				type InterfaceA interface{}
			`,
			expectedMap: types.DeclarationMap{
				"StructA": "testpkg",
			},
		},
		{
			name: "Structs in different package name",
			code: `
				package otherpkg
				type Something struct {}
			`,
			expectedMap: types.DeclarationMap{
				"Something": "otherpkg",
			},
		},
	}

	parser := &VertexParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := parseGoFile(t, tt.code)
			result := parser.parseStructDelcarations(node)
			assert.Equal(t, tt.expectedMap, result)
		})
	}
}

func TestParseReceiver(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		typeMap      types.DeclarationMap
		wantTypeName string
		wantStruct   string
		wantMethod   bool
	}{
		{
			name: "Free function (no receiver)",
			code: `
				func DoSomething() {}
			`,
			typeMap:      types.DeclarationMap{},
			wantTypeName: "",
			wantStruct:   "",
			wantMethod:   false,
		},
		{
			name: "Method with value receiver",
			code: `
				type MyStruct struct{}
				func (m MyStruct) DoSomething() {}
			`,
			typeMap:      types.DeclarationMap{},
			wantTypeName: "MyStruct",
			wantStruct:   "MyStruct",
			wantMethod:   true,
		},
		{
			name: "Method with pointer receiver",
			code: `
				type MyStruct struct{}
				func (m *MyStruct) DoSomething() {}
			`,
			typeMap:      types.DeclarationMap{},
			wantTypeName: "*MyStruct",
			wantStruct:   "MyStruct",
			wantMethod:   true,
		},
		{
			name: "Method with package-mapped receiver",
			code: `
				type MyStruct struct{}
				func (m *MyStruct) DoSomething() {}
			`,
			typeMap:      types.DeclarationMap{"MyStruct": "pkgX"},
			wantTypeName: "*pkgX.MyStruct",
			wantStruct:   "MyStruct",
			wantMethod:   true,
		},
	}

	parser := &VertexParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFunctionCode(t, tt.code)
			typeName, structName, isMethod := parser.parseReceiver(fn, tt.typeMap)

			assert.Equal(t, tt.wantTypeName, typeName)
			assert.Equal(t, tt.wantStruct, structName)
			assert.Equal(t, tt.wantMethod, isMethod)
		})
	}
}

func TestParseFunction(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		structsMap   types.DeclarationMap
		packageName  string
		expectedFunc *types.FunctionInfo
	}{
		{
			name: "Function with no annotation is ignored",
			code: `
				func GetUser() string { return "" }
			`,
			structsMap:   types.DeclarationMap{},
			packageName:  "testpkg",
			expectedFunc: nil,
		},
		{
			name: "Free function with annotation",
			code: `
				// @server path=/users method=GET
				func GetUser() string { return "" }
			`,
			structsMap:  types.DeclarationMap{},
			packageName: "testpkg",
			expectedFunc: &types.FunctionInfo{
				Name:        "GetUser",
				Path:        "/users",
				Method:      "GET",
				Params:      nil,
				ReturnType:  "string",
				IsSlice:     false,
				IsMethod:    false,
				PackageName: "testpkg",
			},
		},
		{
			name: "Method with pointer receiver and slice return",
			code: `
				type User struct{}
				// @server path=/users method=GET
				func (u *User) List() []string { return nil }
			`,
			structsMap:  types.DeclarationMap{"User": "testpkg"},
			packageName: "testpkg",
			expectedFunc: &types.FunctionInfo{
				Name:             "List",
				Path:             "/users",
				Method:           "GET",
				Params:           nil,
				ReturnType:       "[]string",
				IsSlice:          true,
				IsMethod:         true,
				ReceiverTypeName: "*testpkg.User",
				StructName:       "User",
				PackageName:      "testpkg",
			},
		},
		{
			name: "Function with parameters and custom types",
			code: `
				// @server path=/items method=POST
				func CreateUser(name string, age int, meta pkg.Meta) bool { return true }
			`,
			structsMap:  types.DeclarationMap{"Meta": "pkg"},
			packageName: "testpkg",
			expectedFunc: &types.FunctionInfo{
				Name:       "CreateUser",
				Path:       "/items",
				Method:     "POST",
				IsMethod:   false,
				IsSlice:    false,
				ReturnType: "bool",
				Params: []types.ParamInfo{
					{Name: "name", Type: "string"},
					{Name: "age", Type: "int"},
					{Name: "meta", Type: "pkg.Meta"},
				},
				PackageName: "testpkg",
			},
		},
		{
			name: "Method with value receiver and no return",
			code: `
				type Controller struct{}
				// @server path=/items method=POST
				func (c Controller) Clear() {}
			`,
			structsMap:  types.DeclarationMap{"Controller": "testpkg"},
			packageName: "testpkg",
			expectedFunc: &types.FunctionInfo{
				Name:             "Clear",
				Path:             "/items",
				Method:           "POST",
				IsMethod:         true,
				ReceiverTypeName: "testpkg.Controller",
				StructName:       "Controller",
				Params:           nil,
				ReturnType:       "",
				IsSlice:          false,
				PackageName:      "testpkg",
			},
		},
	}

	parser := &VertexParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFunctionCode(t, tt.code)
			result := parser.parseFunction(fn, tt.structsMap, tt.packageName)

			assert.Equal(t, tt.expectedFunc, result)
		})
	}
}

func TestParseFunctions(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		structsMap types.DeclarationMap
		expected   []types.FunctionInfo
	}{
		{
			name: "No functions in file",
			code: `
				var x = 10
			`,
			structsMap: types.DeclarationMap{},
			expected:   nil,
		},
		{
			name: "Single annotated function",
			code: `
				// @server path=/hello method=GET
				func SayHello() string { return "hi" }
			`,
			structsMap: types.DeclarationMap{},
			expected: []types.FunctionInfo{
				{
					Name:        "SayHello",
					Path:        "/hello",
					Method:      "GET",
					Params:      nil,
					ReturnType:  "string",
					IsSlice:     false,
					IsMethod:    false,
					PackageName: "testpkg",
				},
			},
		},
		{
			name: "Mixed: one with comment, one without",
			code: `
				func Unannotated() {}

				// @server path=/data method=POST
				func SaveData(data string) error { return nil }
			`,
			structsMap: types.DeclarationMap{},
			expected: []types.FunctionInfo{
				{
					Name:   "SaveData",
					Path:   "/data",
					Method: "POST",
					Params: []types.ParamInfo{
						{Name: "data", Type: "string"},
					},
					ReturnType:  "error",
					IsSlice:     false,
					IsMethod:    false,
					PackageName: "testpkg",
				},
			},
		},
		{
			name: "Method with pointer receiver",
			code: `
				type Service struct{}
				// @server path=/status method=GET
				func (s *Service) Status() string { return "ok" }
			`,
			structsMap: types.DeclarationMap{"Service": "testpkg"},
			expected: []types.FunctionInfo{
				{
					Name:             "Status",
					Path:             "/status",
					Method:           "GET",
					Params:           nil,
					ReturnType:       "string",
					IsSlice:          false,
					IsMethod:         true,
					ReceiverTypeName: "*testpkg.Service",
					StructName:       "Service",
					PackageName:      "testpkg",
				},
			},
		},
		{
			name: "Multiple annotated functions",
			code: `
				// @server path=/one method=GET
				func One() int { return 1 }

				// @server path=/two method=GET
				func Two() int { return 2 }
			`,
			structsMap: types.DeclarationMap{},
			expected: []types.FunctionInfo{
				{
					Name:        "One",
					Path:        "/one",
					Method:      "GET",
					ReturnType:  "int",
					IsSlice:     false,
					IsMethod:    false,
					PackageName: "testpkg",
				},
				{
					Name:        "Two",
					Path:        "/two",
					Method:      "GET",
					ReturnType:  "int",
					IsSlice:     false,
					IsMethod:    false,
					PackageName: "testpkg",
				},
			},
		},
	}

	vp := &VertexParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", "package testpkg\n"+tt.code, parser.ParseComments)
			require.NoError(t, err)

			result := vp.parseFunctions(node, tt.structsMap)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func parseFunctionCode(t *testing.T, code string) *ast.FuncDecl {
	t.Helper()

	node := parseGoFile(t, "package main\n"+code)
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			return funcDecl
		}
	}

	t.Fatalf("Could not find function declaration in code")
	return nil
}

func parseGoFile(t *testing.T, src string) *ast.File {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	assert.NoError(t, err, "Failed to parse Go code")

	return node
}

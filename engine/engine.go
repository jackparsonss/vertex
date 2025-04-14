package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"

	"github.com/jackparsonss/vertex/internal/constants"
)

type FunctionInfo struct {
	Name       string
	Path       string
	Method     string
	Params     []ParamInfo
	ReturnType string
}

type ParamInfo struct {
	Name string
	Type string
}

func parseFunction(fn *ast.FuncDecl) *FunctionInfo {
	path, method := parseComment(fn)
	if path == "" && method == "" {
		return nil
	}

	params := parseParams(fn)

	var returnType string
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		returnType = fmt.Sprintf("%s", fn.Type.Results.List[0].Type)
	}

	return &FunctionInfo{
		Name:       fn.Name.Name,
		Path:       path,
		Method:     method,
		Params:     params,
		ReturnType: returnType,
	}
}

func parseParams(fn *ast.FuncDecl) []ParamInfo {
	var params []ParamInfo
	if fn.Type.Params == nil {
		return params
	}

	for i, param := range fn.Type.Params.List {
		paramType := fmt.Sprintf("%s", param.Type)
		if len(param.Names) == 0 {
			params = append(params, ParamInfo{Name: fmt.Sprintf("param%d", i), Type: paramType})
			continue
		}

		for _, name := range param.Names {
			params = append(params, ParamInfo{Name: name.Name, Type: paramType})
		}
	}

	return params
}

func parseComment(fn *ast.FuncDecl) (string, string) {
	var path, method string
	for _, comment := range fn.Doc.List {
		text := comment.Text

		if !strings.Contains(text, constants.SERVER_DIRECTIVE) {
			continue
		}

		if strings.Contains(text, constants.PATH_DIRECTIVE) {
			pathStart := strings.Index(text, constants.PATH_DIRECTIVE) + 5
			pathEnd := strings.Index(text[pathStart:], " ")
			if pathEnd == -1 {
				path = text[pathStart:]
			} else {
				path = text[pathStart : pathStart+pathEnd]
			}
		}

		if strings.Contains(text, constants.METHOD_DIRECTIVE) {
			methodStart := strings.Index(text, constants.METHOD_DIRECTIVE) + 7
			methodEnd := strings.Index(text[methodStart:], " ")
			if methodEnd == -1 {
				method = text[methodStart:]
			} else {
				method = text[methodStart : methodStart+methodEnd]
			}
		}
	}

	return path, method
}

func parseFunctions(node *ast.File) []FunctionInfo {
	var functions []FunctionInfo
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if fn.Doc == nil {
			return true
		}

		f := parseFunction(fn)
		if f == nil {
			return true
		}

		functions = append(functions, *f)

		return true
	})

	return functions
}

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "../cmd/main.go", nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll("../generated", 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	functions := parseFunctions(node)

	generateServerCode(functions)
	generateClientCode(functions)
}

func generateServerCode(functions []FunctionInfo) {
	tmpl := template.Must(template.New("server.tmpl").ParseFiles("../engine/server.tmpl"))

	file, err := os.Create("../generated/server.go")
	if err != nil {
		fmt.Printf("Error creating server file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, functions); err != nil {
		fmt.Printf("Error executing server template: %v\n", err)
		os.Exit(1)
	}
}

func generateClientCode(functions []FunctionInfo) {
	tmpl := template.Must(template.New("client.tmpl").ParseFiles("../engine/client.tmpl"))

	file, err := os.Create("../generated/client.go")
	if err != nil {
		fmt.Printf("Error creating client file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, functions); err != nil {
		fmt.Printf("Error executing client template: %v\n", err)
		os.Exit(1)
	}
}

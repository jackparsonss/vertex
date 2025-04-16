package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"

	"github.com/jackparsonss/vertex/engine/codegen"
	"github.com/jackparsonss/vertex/internal/constants"
)

type FunctionInfo struct {
	Name             string
	Path             string
	Method           string
	Params           []ParamInfo
	ReturnType       string
	IsStruct         bool
	IsSlice          bool
	IsMethod         bool
	ReceiverTypeName string
	StructName       string
	PackageName      string
}

type ParamInfo struct {
	Name string
	Type string
}

func parseStructDelcarations(node *ast.File) map[string]bool {
	structs := make(map[string]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if _, ok := typeSpec.Type.(*ast.StructType); ok {
			structs[typeSpec.Name.Name] = true
		}
		return true
	})

	return structs
}

func parseReceiver(fn *ast.FuncDecl, packageName string) (string, string, bool) {
	isMethod := fn.Recv != nil && len(fn.Recv.List) > 0
	if !isMethod {
		return "", "", false
	}

	var receiverTypeName, structName string
	receiverExpr := fn.Recv.List[0].Type
	receiverTypeName = codegen.GetTypeString(receiverExpr, packageName)

	// Extract the struct name without pointer if it's a pointer
	if starExpr, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
		if ident, ok := starExpr.X.(*ast.Ident); ok {
			structName = ident.Name
		}
	} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
		structName = ident.Name
	}

	return receiverTypeName, structName, isMethod
}

func parseFunction(fn *ast.FuncDecl, structsMap map[string]bool, packageName string) *FunctionInfo {
	path, method := parseComment(fn)
	if path == "" && method == "" {
		return nil
	}

	receiverTypeName, structName, isMethod := parseReceiver(fn, packageName)
	params := parseParams(fn)
	returnType, isStruct, isSlice := parseReturnType(fn, structsMap, packageName)

	return &FunctionInfo{
		Name:             fn.Name.Name,
		Path:             path,
		Method:           method,
		Params:           params,
		ReturnType:       returnType,
		IsStruct:         isStruct,
		IsSlice:          isSlice,
		ReceiverTypeName: receiverTypeName,
		StructName:       structName,
		IsMethod:         isMethod,
		PackageName:      packageName,
	}
}

func parseReturnType(fn *ast.FuncDecl, structsMap map[string]bool, packageName string) (string, bool, bool) {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return "", false, false
	}

	var isStruct, isSlice bool
	_, isStruct = fn.Type.Results.List[0].Type.(*ast.StructType)
	if ident, ok := fn.Type.Results.List[0].Type.(*ast.Ident); ok {
		if _, exists := structsMap[ident.Name]; exists {
			isStruct = true
		}
	}

	if starExpr, ok := fn.Type.Results.List[0].Type.(*ast.StarExpr); ok {
		if ident, ok := starExpr.X.(*ast.Ident); ok {
			if _, exists := structsMap[ident.Name]; exists {
				isStruct = true
			}
		}
	}

	if arrayType, ok := fn.Type.Results.List[0].Type.(*ast.ArrayType); ok {
		isSlice = true
		if ident, ok := arrayType.Elt.(*ast.Ident); ok {
			if _, exists := structsMap[ident.Name]; exists {
				isStruct = true
			}
		}
	}

	return codegen.GetTypeString(fn.Type.Results.List[0].Type, packageName), isStruct, isSlice
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

func parseFunctions(node *ast.File, structsMap map[string]bool) []FunctionInfo {
	packageName := node.Name.Name

	var functions []FunctionInfo
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if fn.Doc == nil {
			return true
		}

		f := parseFunction(fn, structsMap, packageName)
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
	node, err := parser.ParseFile(fset, "../../cmd/product/product.go", nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll("../generated", 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	structs := parseStructDelcarations(node)
	functions := parseFunctions(node, structs)

	generateServerCode(functions)
	generateClientCode(functions)
}

func generateServerCode(functions []FunctionInfo) {
	tmpl := template.Must(template.New("server.tmpl").ParseFiles("../../engine/templates/server.tmpl"))

	packageName := ""
	if len(functions) > 0 {
		packageName = functions[0].PackageName
	}

	structFuncs := make(map[string][]FunctionInfo)
	var standaloneFuncs []FunctionInfo

	for _, fn := range functions {
		if fn.IsMethod {
			structFuncs[fn.StructName] = append(structFuncs[fn.StructName], fn)
		} else {
			standaloneFuncs = append(standaloneFuncs, fn)
		}
	}

	allFunctions := make([]FunctionInfo, 0)
	for _, fns := range structFuncs {
		allFunctions = append(allFunctions, fns...)
	}
	allFunctions = append(allFunctions, standaloneFuncs...)

	templateData := struct {
		PackageName     string
		StructFuncs     map[string][]FunctionInfo
		StandaloneFuncs []FunctionInfo
		AllFunctions    []FunctionInfo
	}{
		PackageName:     packageName,
		StructFuncs:     structFuncs,
		StandaloneFuncs: standaloneFuncs,
		AllFunctions:    allFunctions,
	}

	file, err := os.Create("../generated/server.go")
	if err != nil {
		fmt.Printf("Error creating server file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, templateData); err != nil {
		fmt.Printf("Error executing server template: %v\n", err)
		os.Exit(1)
	}
}

func generateClientCode(functions []FunctionInfo) {
	tmpl := template.Must(template.New("client.tmpl").ParseFiles("../../engine/templates/client.tmpl"))

	packageName := ""
	if len(functions) > 0 {
		packageName = functions[0].PackageName
	}

	templateData := struct {
		PackageName string
		Functions   []FunctionInfo
	}{
		PackageName: packageName,
		Functions:   functions,
	}

	file, err := os.Create("../generated/client.go")
	if err != nil {
		fmt.Printf("Error creating client file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, templateData); err != nil {
		fmt.Printf("Error executing client template: %v\n", err)
		os.Exit(1)
	}
}

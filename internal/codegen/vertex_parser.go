package codegen

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/jackparsonss/vertex/internal/constants"
)

type ParamInfo struct {
	Name string
	Type string
}

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

type DeclarationMap map[string]bool

type VertexParser struct {
	node *ast.File
}

func NewVertexParser(node *ast.File) *VertexParser {
	return &VertexParser{node: node}
}

func (v *VertexParser) Parse() []FunctionInfo {
	structs := v.parseStructDelcarations()
	functions := v.parseFunctions(structs)

	return functions
}

func (v *VertexParser) parseStructDelcarations() DeclarationMap {
	structs := make(DeclarationMap)
	ast.Inspect(v.node, func(n ast.Node) bool {
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

func (v *VertexParser) parseReceiver(fn *ast.FuncDecl, packageName string) (string, string, bool) {
	isMethod := fn.Recv != nil && len(fn.Recv.List) > 0
	if !isMethod {
		return "", "", false
	}

	var receiverTypeName, structName string
	receiverExpr := fn.Recv.List[0].Type
	receiverTypeName = GetTypeString(receiverExpr, packageName)

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

func (v *VertexParser) parseFunction(fn *ast.FuncDecl, structsMap DeclarationMap, packageName string) *FunctionInfo {
	path, method := v.parseComment(fn)
	if path == "" && method == "" {
		return nil
	}

	receiverTypeName, structName, isMethod := v.parseReceiver(fn, packageName)
	params := v.parseParams(fn)
	returnType, isStruct, isSlice := v.parseReturnType(fn, structsMap, packageName)

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

func (v *VertexParser) parseParams(fn *ast.FuncDecl) []ParamInfo {
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

func (v *VertexParser) parseComment(fn *ast.FuncDecl) (string, string) {
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

func (v *VertexParser) parseFunctions(structsMap DeclarationMap) []FunctionInfo {
	packageName := v.node.Name.Name

	var functions []FunctionInfo
	ast.Inspect(v.node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if fn.Doc == nil {
			return true
		}

		f := v.parseFunction(fn, structsMap, packageName)
		if f == nil {
			return true
		}

		functions = append(functions, *f)

		return true
	})

	return functions
}

func (v *VertexParser) parseReturnType(fn *ast.FuncDecl, structsMap DeclarationMap, packageName string) (string, bool, bool) {
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

	return GetTypeString(fn.Type.Results.List[0].Type, packageName), isStruct, isSlice
}

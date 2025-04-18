package parser

import (
	"bufio"
	"fmt"
	"go/ast"
	"os"
	"strings"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/jackparsonss/vertex/internal/codegen/utils"
	"github.com/jackparsonss/vertex/internal/config"
	"github.com/jackparsonss/vertex/internal/constants"
)

type DeclarationMap map[string]bool

type VertexParser struct {
	nodes  []*ast.File
	config config.Config
}

func NewVertexParser(nodes []*ast.File, config config.Config) *VertexParser {
	return &VertexParser{nodes: nodes, config: config}
}

func (v *VertexParser) Parse() (types.Vertex, error) {
	goModPackage, err := v.ParseGoMod(v.config.GoModFile)
	functions := []types.FunctionInfo{}

	for _, node := range v.nodes {
		structs := v.parseStructDelcarations(node)
		functions = append(functions, v.parseFunctions(node, structs)...)
	}
	if err != nil {
		return types.Vertex{}, err
	}

	return types.Vertex{
		GoModPackage: goModPackage,
		Functions:    functions,
	}, nil
}

func (v *VertexParser) ParseGoMod(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			return moduleName, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no module declaration found in %s", path)
}

func (v *VertexParser) parseStructDelcarations(node *ast.File) DeclarationMap {
	structs := make(DeclarationMap)
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

func (v *VertexParser) parseReceiver(fn *ast.FuncDecl, packageName string) (string, string, bool) {
	isMethod := fn.Recv != nil && len(fn.Recv.List) > 0
	if !isMethod {
		return "", "", false
	}

	var receiverTypeName, structName string
	receiverExpr := fn.Recv.List[0].Type
	receiverTypeName = utils.GetTypeString(receiverExpr, packageName)

	if starExpr, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
		if ident, ok := starExpr.X.(*ast.Ident); ok {
			structName = ident.Name
		}
	} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
		structName = ident.Name
	}

	return receiverTypeName, structName, isMethod
}

func (v *VertexParser) parseFunction(fn *ast.FuncDecl, structsMap DeclarationMap, packageName string) *types.FunctionInfo {
	path, method := v.parseComment(fn)
	if path == "" && method == "" {
		return nil
	}

	receiverTypeName, structName, isMethod := v.parseReceiver(fn, packageName)
	params := v.parseParams(fn)
	returnType, isStruct, isSlice := v.parseReturnType(fn, structsMap, packageName)

	return &types.FunctionInfo{
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

func (v *VertexParser) parseParams(fn *ast.FuncDecl) []types.ParamInfo {
	var params []types.ParamInfo
	if fn.Type.Params == nil {
		return params
	}

	for i, param := range fn.Type.Params.List {
		paramType := fmt.Sprintf("%s", param.Type)
		if len(param.Names) == 0 {
			params = append(params, types.ParamInfo{Name: fmt.Sprintf("param%d", i), Type: paramType})
			continue
		}

		for _, name := range param.Names {
			params = append(params, types.ParamInfo{Name: name.Name, Type: paramType})
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

func (v *VertexParser) parseFunctions(node *ast.File, structsMap DeclarationMap) []types.FunctionInfo {
	packageName := node.Name.Name

	var functions []types.FunctionInfo
	ast.Inspect(node, func(n ast.Node) bool {
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

	return utils.GetTypeString(fn.Type.Results.List[0].Type, packageName), isStruct, isSlice
}

package parser

import (
	"bufio"
	"fmt"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackparsonss/vertex/internal/codegen/types"
	"github.com/jackparsonss/vertex/internal/codegen/utils"
	"github.com/jackparsonss/vertex/internal/config"
	"github.com/jackparsonss/vertex/internal/constants"
)

type VertexParser struct {
	nodes  []*ast.File
	config config.Config
}

func NewVertexParser(nodes []*ast.File, config config.Config) *VertexParser {
	return &VertexParser{nodes: nodes, config: config}
}

func (v *VertexParser) Parse() (types.Vertex, error) {
	goModPackage, err := v.ParseGoMod(v.config.GoModFile)
	if err != nil {
		return types.Vertex{}, err
	}

	err = v.AddGoModReplace(v.config.GoModFile)
	if err != nil {
		return types.Vertex{}, err
	}

	err = v.RunGoModTidy(v.config.GoModFile)
	if err != nil {
		return types.Vertex{}, err
	}

	functions := []types.FunctionInfo{}
	for _, node := range v.nodes {
		structs := v.parseStructDelcarations(node)
		functions = append(functions, v.parseFunctions(node, structs)...)
	}

	return types.Vertex{
		GoModPackage: goModPackage,
		Functions:    functions,
	}, nil
}

func (v *VertexParser) AddGoModReplace(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	contentStr := string(content)

	replacePattern := regexp.MustCompile(`(?m)^replace\s+vertex\s*=>\s*\./vertex`)
	if replacePattern.MatchString(contentStr) {
		return nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = file.WriteString("\nreplace vertex => ./vertex\n")
	return err
}

func (v *VertexParser) RunGoModTidy(path string) error {
	dir := filepath.Dir(path)

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %v\nOutput: %s", err, output)
	}

	return nil
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

func (v *VertexParser) parseStructDelcarations(node *ast.File) types.DeclarationMap {
	structs := make(types.DeclarationMap)
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if _, ok := typeSpec.Type.(*ast.StructType); ok {
			structs[typeSpec.Name.Name] = node.Name.Name
		}
		return true
	})

	return structs
}

func (v *VertexParser) parseReceiver(fn *ast.FuncDecl, packageName types.DeclarationMap) (string, string, bool) {
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

func (v *VertexParser) parseFunction(fn *ast.FuncDecl, structsMap types.DeclarationMap, packageName string) *types.FunctionInfo {
	path, method := v.parseComment(fn)
	if path == "" && method == "" {
		return nil
	}

	receiverTypeName, structName, isMethod := v.parseReceiver(fn, structsMap)
	params := v.parseParams(fn)
	returnType, isSlice := v.parseReturnType(fn, structsMap)

	return &types.FunctionInfo{
		Name:             fn.Name.Name,
		Path:             path,
		Method:           method,
		Params:           params,
		ReturnType:       returnType,
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
	if fn.Doc == nil {
		return "", ""
	}

	var path, method string
	for _, comment := range fn.Doc.List {
		text := comment.Text
		if !strings.Contains(text, constants.SERVER_DIRECTIVE) {
			continue
		}

		pathDirective := strings.TrimSuffix(constants.PATH_DIRECTIVE, "=")
		methodDirective := strings.TrimSuffix(constants.METHOD_DIRECTIVE, "=")

		pathPattern := regexp.MustCompile(pathDirective + `\s*=\s*(\S+)`)
		pathMatches := pathPattern.FindStringSubmatch(text)
		if len(pathMatches) > 1 {
			path = pathMatches[1]
		}

		if path == "" {
			return "", ""
		}

		methodPattern := regexp.MustCompile(methodDirective + `\s*=\s*(\S+)`)
		methodMatches := methodPattern.FindStringSubmatch(text)
		if len(methodMatches) > 1 {
			method = methodMatches[1]
		}

		if method == "" {
			return "", ""
		}
	}

	return path, method
}

func (v *VertexParser) parseFunctions(node *ast.File, structsMap types.DeclarationMap) []types.FunctionInfo {
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

func (v *VertexParser) parseReturnType(fn *ast.FuncDecl, structsMap types.DeclarationMap) (string, bool) {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return "", false
	}

	isSlice := false
	if _, ok := fn.Type.Results.List[0].Type.(*ast.ArrayType); ok {
		isSlice = true
	}

	return utils.GetTypeString(fn.Type.Results.List[0].Type, structsMap), isSlice
}

package utils

import (
	"fmt"
	"go/ast"

	"github.com/jackparsonss/vertex/internal/codegen/types"
)

func GetTypeString(expr ast.Expr, typeMap types.DeclarationMap) string {
	switch t := expr.(type) {
	case *ast.Ident:
		prefix := ""
		if packageName, ok := typeMap[t.Name]; ok {
			prefix = packageName + "."
		}
		return prefix + t.Name
	case *ast.StarExpr:
		return "*" + GetTypeString(t.X, typeMap)
	case *ast.SelectorExpr:
		return GetTypeString(t.X, typeMap) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + GetTypeString(t.Elt, typeMap)
	case *ast.MapType:
		return "map[" + GetTypeString(t.Key, typeMap) + "]" + GetTypeString(t.Value, typeMap)
	case *ast.StructType:
		return "struct{}"
	case *ast.FuncType:
		return formatFuncType(t, typeMap)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + GetTypeString(t.Elt, typeMap)
	case *ast.ChanType:
		return "chan " + GetTypeString(t.Value, typeMap)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func formatFuncType(t *ast.FuncType, typeMap types.DeclarationMap) string {
	result := "func("

	if t.Params != nil && len(t.Params.List) > 0 {
		for i, param := range t.Params.List {
			if i > 0 {
				result += ", "
			}
			result += GetTypeString(param.Type, typeMap)
		}
	}

	result += ")"

	if t.Results == nil || len(t.Results.List) == 0 {
		return result
	}

	if len(t.Results.List) == 1 && len(t.Results.List[0].Names) == 0 {
		return result + " " + GetTypeString(t.Results.List[0].Type, typeMap)
	}

	result += " ("
	for i, ret := range t.Results.List {
		if i > 0 {
			result += ", "
		}
		result += GetTypeString(ret.Type, typeMap)
	}
	result += ")"

	return result
}

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
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

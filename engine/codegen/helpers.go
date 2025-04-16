package codegen

import (
	"fmt"
	"go/ast"
)

func GetTypeString(expr ast.Expr, packageName string) string {
	prefix := ""
	if packageName != "" {
		prefix = packageName + "."
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return prefix + t.Name
	case *ast.StarExpr:
		return "*" + GetTypeString(t.X, packageName)
	case *ast.SelectorExpr:
		return GetTypeString(t.X, packageName) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + GetTypeString(t.Elt, packageName)
	case *ast.MapType:
		return "map[" + GetTypeString(t.Key, packageName) + "]" + prefix + GetTypeString(t.Value, packageName)
	case *ast.StructType:
		return "struct{}"
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

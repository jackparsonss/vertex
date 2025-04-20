package utils

import (
	"fmt"
	"go/ast"
	"go/token"

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
		if t.Len == nil {
			return "[]" + GetTypeString(t.Elt, typeMap)
		}

		if lit, ok := t.Len.(*ast.BasicLit); ok && lit.Kind == token.INT {
			return "[" + lit.Value + "]" + GetTypeString(t.Elt, typeMap)
		}

		return "[" + GetTypeString(t.Len, typeMap) + "]" + GetTypeString(t.Elt, typeMap)
	case *ast.MapType:
		return "map[" + GetTypeString(t.Key, typeMap) + "]" + GetTypeString(t.Value, typeMap)
	case *ast.StructType:
		if t.Fields != nil && len(t.Fields.List) > 0 {
			result := "struct{"
			for i, field := range t.Fields.List {
				if i > 0 {
					result += "\n"
				}

				if len(field.Names) > 0 {
					for j, name := range field.Names {
						if j > 0 {
							result += ", "
						}
						result += name.Name
					}
					result += " "
				}

				result += GetTypeString(field.Type, typeMap)

				if field.Tag != nil {
					result += " " + field.Tag.Value
				}
			}
			result += "}"
			return result
		}
		return "struct{}"
	case *ast.FuncType:
		return "func" + formatFuncSig(t, typeMap)
	case *ast.InterfaceType:
		if t.Methods != nil && len(t.Methods.List) > 0 {
			result := "interface{"
			for i, method := range t.Methods.List {
				if i > 0 {
					result += "\n"
				}

				if len(method.Names) > 0 {
					result += method.Names[0].Name
				}

				if ft, ok := method.Type.(*ast.FuncType); ok {
					result += formatFuncSig(ft, typeMap)
				} else {
					result += " " + GetTypeString(method.Type, typeMap)
				}
			}
			result += "}"
			return result
		}
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + GetTypeString(t.Elt, typeMap)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + GetTypeString(t.Value, typeMap)
		case ast.RECV:
			return "<-chan " + GetTypeString(t.Value, typeMap)
		default:
			return "chan " + GetTypeString(t.Value, typeMap)
		}
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func formatFuncSig(t *ast.FuncType, typeMap types.DeclarationMap) string {
	result := "("

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

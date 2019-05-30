package main

import (
	"go/ast"
)

func valueOf(x ast.Node) string {
	switch x := x.(type) {
	case *ast.BasicLit:
		return x.Value
	case *ast.Ident:
		return x.Name
	default:
		return ""
	}
}

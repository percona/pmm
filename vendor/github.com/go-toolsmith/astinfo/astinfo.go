// Package astinfo records useful AST information like node parents and such.
package astinfo

import (
	"go/ast"
)

// Info holds AST metadata collected during Origin field traversal.
type Info struct {
	// Origin is a node that was used to collect all the
	// information stored inside this object.
	Origin ast.Node

	// All non-nil maps below are populeted during Resolve method execution.
	// Nil maps remain nil.
	// This is the same behavior that you get with types.Info.

	// Parents maps child AST not to its parent.
	//
	// Does not contain top-level nodes, so the lookup will
	// return nil for those.
	Parents map[ast.Node]ast.Node
}

// Resolve fills AST info collected during info.Origin traversal.
func (info *Info) Resolve() {
	if info.Origin == nil {
		return
	}

	info.clearMaps()

	var parents []ast.Node

	ast.Inspect(info.Origin, func(x ast.Node) bool {
		if x == nil {
			parents = parents[:len(parents)-1]
			return false
		}

		if info.Parents != nil && len(parents) != 0 {
			info.Parents[x] = parents[len(parents)-1]
		}

		parents = append(parents, x)

		return true
	})

	if len(parents) != 0 {
		panic("unexpected non-empty parents after traversal")
	}
}

func (info *Info) clearMaps() {
	for k := range info.Parents {
		delete(info.Parents, k)
	}
}

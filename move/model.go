package move

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/podhmo/gomvpkg-light/collect"
)

// Req :
type Req struct {
	FromPkg     string
	ToPkg       string
	InPkg       string
	Root        *collect.Target
	Affected    []collect.Affected
	WillBeWrite map[*token.File]*PreWrite
}

// PreWrite :
type PreWrite struct {
	Pkg  *types.Package
	File *ast.File
}

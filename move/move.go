package move

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/pkg/errors"
	"github.com/podhmo/gomvpkg-light/collect"
	"golang.org/x/tools/go/loader"
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

// TargetPackage :
func TargetPackage(prog *loader.Program, req *Req) error {
	src := prog.Package(req.FromPkg)
	if src == nil {
		return errors.Errorf("not found pkg %s", req.FromPkg)
	}

	dst := prog.Package(req.ToPkg)
	var pkgname string
	if dst != nil {
		pkgname = dst.Pkg.Name()
	} else {
		elems := strings.Split(req.ToPkg, "/")
		pkgname = elems[len(elems)-1]
	}

	for _, f := range src.Files {
		f := f
		f.Name.Name = pkgname
		k := prog.Fset.File(f.Pos())
		req.WillBeWrite[k] = &PreWrite{
			Pkg:  src.Pkg,
			File: f,
		}
	}
	return nil
}

// AffectedPackages :
func AffectedPackages(prog *loader.Program, req *Req) error {
	return nil
}

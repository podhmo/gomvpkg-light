package move

import (
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

// TargetPackage :
func TargetPackage(prog *loader.Program, req *Req) error {
	from := prog.Package(req.FromPkg)
	if from == nil {
		return errors.Errorf("not found pkg %s", req.FromPkg)
	}

	to := prog.Package(req.ToPkg)
	var pkgname string
	if to != nil {
		pkgname = to.Pkg.Name()
	} else {
		elems := strings.Split(req.ToPkg, "/")
		pkgname = elems[len(elems)-1]
	}

	for _, f := range from.Files {
		f := f
		f.Name.Name = pkgname
		k := prog.Fset.File(f.Pos())
		req.WillBeWrite[k] = &PreWrite{
			Pkg:  from.Pkg,
			File: f,
		}
	}
	return nil
}

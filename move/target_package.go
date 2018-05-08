package move

import (
	"go/token"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

var (
	importCommentRX = regexp.MustCompile(`import +"[^"]+"`)
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

		// Update all import comments.
		for _, cg := range f.Comments {
			c := cg.List[0]
			if c.Slash >= f.Name.End() &&
				sameLine(prog.Fset, c.Slash, f.Name.End()) &&
				(f.Decls == nil || c.Slash < f.Decls[0].Pos()) {
				if strings.HasPrefix(c.Text, `// import "`) {
					c.Text = `// import "` + req.ToPkg + `"`
					break
				}
				if strings.HasPrefix(c.Text, `/* import "`) {
					c.Text = `/* import "` + req.ToPkg + `" */`
					break
				}
			}
		}
	}
	return nil
}

// sameLine reports whether two positions in the same file are on the same line.
func sameLine(fset *token.FileSet, x, y token.Pos) bool {
	return fset.Position(x).Line == fset.Position(y).Line
}

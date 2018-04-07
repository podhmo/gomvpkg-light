package move

import (
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/podhmo/gomvpkg-light/collect"
	"golang.org/x/tools/go/ast/astutil"
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

// AffectedPackages :
func AffectedPackages(prog *loader.Program, req *Req) error {
	/*
		memo(TODO):
				- unnamed import -> need to replace
				- named import -> don't need to replace
				- named import, but same name of calculated from frompkg -> replace
				- another same name import is existed -> conflict(error?)
				- another same name import is existed (named import) -> conflict
	*/
	frominfo := prog.Package(req.FromPkg)
	if frominfo == nil {
		return errors.Errorf("not found pkg %s", req.FromPkg)
	}
	frompkg := frominfo.Pkg

	var topkg *types.Package
	toinfo := prog.Package(req.ToPkg)
	if toinfo == nil {
		elems := strings.Split(req.ToPkg, "/")
		topkg = types.NewPackage(req.ToPkg, elems[len(elems)-1])
	} else {
		topkg = toinfo.Pkg
	}

	m := &mover{
		prog:    prog,
		req:     req,
		frompkg: frompkg,
		topkg:   topkg,
	}
	for _, a := range req.Affected {
		if err := m.apply(&a); err != nil {
			return err
		}
	}
	return nil
}

type mover struct {
	prog    *loader.Program
	req     *Req
	frompkg *types.Package
	topkg   *types.Package
}

func (m *mover) apply(a *collect.Affected) error {
	info := m.prog.Package(a.Pkg)
	if info == nil {
		return errors.Errorf("package not found %v", a.Pkg)
	}

	fset := m.prog.Fset
	fileMap := map[string]*ast.File{}
	for _, f := range info.Files {
		f := f
		name := filepath.Base(fset.File(f.Pos()).Name())
		fileMap[name] = f
	}

	for _, fname := range a.Files {
		f, ok := fileMap[fname]
		if !ok {
			log.Printf("%s/%s is not found", a.Pkg, fname)
			continue
		}

		importName := m.frompkg.Name()
		for _, is := range f.Imports {
			path, err := strconv.Unquote(is.Path.Value)
			if err != nil {
				return errors.Errorf("invalid path %s, in %q", err, fname)
			}
			if path == m.frompkg.Path() {
				if is.Name != nil {
					importName = is.Name.Name
				}
			}
		}

		skip := false

		if importName != m.frompkg.Name() {
			skip = true
		}

		// todo : check
		if !skip {
			ast.Inspect(f, func(node ast.Node) bool {
				if t, _ := node.(*ast.SelectorExpr); t != nil {
					if m.frompkg == info.ObjectOf(t.Sel).Pkg() {
						ast.Inspect(t.X, func(node ast.Node) bool {
							if ident, _ := node.(*ast.Ident); ident != nil {
								ident.Name = m.topkg.Name()
							}
							return true
						})
						return false
					}
				}
				return true
			})

		}
		astutil.RewriteImport(fset, f, m.frompkg.Path(), m.topkg.Path())

		k := fset.File(f.Pos())
		m.req.WillBeWrite[k] = &PreWrite{
			Pkg:  info.Pkg,
			File: f,
		}
	}
	return nil
}

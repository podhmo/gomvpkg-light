package move

import (
	"go/ast"
	"go/types"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/podhmo/gomvpkg-light/build"
	"github.com/podhmo/gomvpkg-light/collect"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

// AffectedPackages :
func AffectedPackages(ctxt *build.Context, prog *loader.Program, req *Req) error {
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
		ctxt:    ctxt,
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
	ctxt    *build.Context
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
		var rewriteImportCandidates []string

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
			if m.ctxt.MatchPkg(m.frompkg.Path(), path) {
				rewriteImportCandidates = append(rewriteImportCandidates, path)
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
					ob := info.ObjectOf(t.Sel)
					if ob == nil {
						log.Printf("affected package, inspect, %q is nil (in %s/%s)", t.Sel, a.Pkg, fname)
						return true
					}
					pkg := ob.Pkg()
					if m.frompkg == pkg || info.Pkg == pkg { // xxx: for embeded (info.Pkg == pkg)
						ast.Inspect(t.X, func(node ast.Node) bool {
							if ident, _ := node.(*ast.Ident); ident != nil {
								if ident.Name == importName && ident.Obj == nil {
									ident.Name = m.topkg.Name()
								}
							}
							return true
						})
						return false
					}
				}
				return true
			})

		}

		for _, path := range rewriteImportCandidates {
			astutil.RewriteImport(fset, f, path, strings.Replace(path, m.frompkg.Path(), m.topkg.Path(), 1))
		}

		k := fset.File(f.Pos())
		m.req.WillBeWrite[k] = &PreWrite{
			Pkg:  info.Pkg,
			File: f,
		}
	}
	return nil
}

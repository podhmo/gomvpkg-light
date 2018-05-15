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
	"github.com/podhmo/gomvpkg-light/build"
	"github.com/podhmo/gomvpkg-light/collect"
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

		seen := map[string][]string{}

		for _, is := range f.Imports {
			var name string

			path, err := strconv.Unquote(is.Path.Value)
			if err != nil {
				return errors.Errorf("invalid path %s, in %q", err, fname)
			}

			if is.Name != nil {
				name = is.Name.Name
				if path == m.frompkg.Path() {
					importName = is.Name.Name
				}
			} else {
				items := strings.Split(path, "/")
				name = items[len(items)-1]
			}

			if m.ctxt.MatchPkg(m.frompkg.Path(), path) {
				rewriteImportCandidates = append(rewriteImportCandidates, path)
				if m.frompkg.Path() == path {
					name = m.topkg.Name()
				}
				seen[name] = append(seen[name], strings.Replace(path, m.frompkg.Path(), m.topkg.Path(), 1))
			} else {
				seen[name] = append(seen[name], path)
			}
		}

		skip := false

		if importName != m.frompkg.Name() {
			skip = true
		}

		if vs, _ := seen[m.topkg.Name()]; len(vs) > 1 {
			log.Printf("conflict: %s in (in %s/%s)", vs, a.Pkg, fname)
			skip = true
		}

		// todo : check
		if !skip {
			ast.Inspect(f, func(node ast.Node) bool {
				if t, _ := node.(*ast.SelectorExpr); t != nil {
					ob := info.ObjectOf(t.Sel)
					if ob == nil {
						if m.req.Verbose {
							log.Printf("affected package, inspect, %q is nil (in %s/%s)", t.Sel, a.Pkg, fname)
						}
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
			rewriteImport(fset, f, path, strings.Replace(path, m.frompkg.Path(), m.topkg.Path(), 1), func(imp *ast.ImportSpec) {
				if imp.Name != nil {
					if imp.Name.Name == importName {
						imp.Name.Name = m.topkg.Name()
					}
				}
			})
		}

		k := fset.File(f.Pos())
		m.req.WillBeWrite[k] = &PreWrite{
			Pkg:  info.Pkg,
			File: f,
		}
	}
	return nil
}

func importPath(s *ast.ImportSpec) string {
	t, err := strconv.Unquote(s.Path.Value)
	if err == nil {
		return t
	}
	return ""
}

// rewriteImport rewrites any import of path oldPath to path newPath.
func rewriteImport(fset *token.FileSet, f *ast.File, oldPath, newPath string, cont func(imp *ast.ImportSpec)) (rewrote bool) {
	for _, imp := range f.Imports {
		if importPath(imp) == oldPath {
			rewrote = true
			// record old End, because the default is to compute
			// it using the length of imp.Path.Value.
			imp.EndPos = imp.End()
			imp.Path.Value = strconv.Quote(newPath)
			cont(imp)
		}
	}
	return
}

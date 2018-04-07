package collect

import (
	"go/parser"
	"go/token"
	"log"
	"strconv"
	"strings"

	"github.com/podhmo/gomvpkg-light/build"
)

// Affected :
type Affected struct {
	Dir   string
	Name  string
	Pkg   string
	Files []string
}

// AffectedPackages :
func AffectedPackages(ctxt *build.Context, srcpkg string, root *Target, pkgdirs []string) ([]Affected, error) {
	var affected []Affected

	fset := token.NewFileSet()
	for _, dir := range pkgdirs {
		fs, err := ctxt.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		item := Affected{
			Dir: dir,
			Pkg: dir[len(root.Dir)+1:],
		}

		for _, f := range fs {
			if !strings.HasSuffix(f.Name(), ".go") {
				continue
			}
			func() {
				r, err := ctxt.OpenFile(ctxt.JoinPath(dir, f.Name()))
				if err != nil {
					log.Println(f.Name(), err)
					return
				}
				defer r.Close()
				astf, err := parser.ParseFile(fset, f.Name(), r, parser.ImportsOnly)
				if err != nil {
					log.Println(f.Name(), err)
					return
				}

				item.Name = astf.Name.Name

				for _, is := range astf.Imports {
					path, err := strconv.Unquote(is.Path.Value)
					if err != nil {
						log.Println(f.Name(), err)
					}
					if path == srcpkg {
						item.Files = append(item.Files, f.Name())
						break
					}
				}
			}()
		}
		if len(item.Files) > 0 {
			affected = append(affected, item)
		}
	}
	return affected, nil
}

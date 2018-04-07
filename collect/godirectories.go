package collect

import (
	"go/build"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/buildutil"
)

// GoFilesDirectories finds go package in inpkg
func GoFilesDirectories(ctxt *build.Context, root *Target) ([]string, error) {
	var pkgdirs []string
	q := []string{root.Path}

	for len(q) > 0 {
		dir := q[0]
		q = q[1:]

		fs, err := buildutil.ReadDir(ctxt, dir)
		if err != nil {
			return nil, errors.Wrap(err, "collect go files directories")
		}

		used := false
		for _, f := range fs {
			if f.IsDir() {
				q = append(q, buildutil.JoinPath(ctxt, dir, f.Name()))
				continue
			}
			if used {
				continue
			}
			if strings.HasSuffix(f.Name(), ".go") {
				pkgdirs = append(pkgdirs, dir)
				used = true
			}
		}
	}
	return pkgdirs, nil
}

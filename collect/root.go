package collect

import (
	"go/build"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/buildutil"
)

// Target :
type Target struct {
	Dir  string
	Pkg  string
	Path string // Dir + Pkg
}

// TargetRoot :
func TargetRoot(ctxt *build.Context, inpkg string) (*Target, error) {
	for _, dir := range ctxt.SrcDirs() {
		path := buildutil.JoinPath(ctxt, dir, inpkg)
		if buildutil.IsDir(ctxt, path) {
			return &Target{
				Dir:  dir,
				Path: path,
				Pkg:  inpkg,
			}, nil
		}
	}
	return nil, errors.Errorf("not found %s", inpkg)
}

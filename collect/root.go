package collect

import (
	"github.com/pkg/errors"
	"github.com/podhmo/gomvpkg-light/build"
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
		path := ctxt.JoinPath(dir, inpkg)
		if ctxt.IsDir(path) {
			return &Target{
				Dir:  dir,
				Path: path,
				Pkg:  inpkg,
			}, nil
		}
	}
	return nil, errors.Errorf("not found %s", inpkg)
}

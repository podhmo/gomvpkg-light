package build

import (
	"go/build"
	"io"
	"os"

	"golang.org/x/tools/go/buildutil"
)

// Default :
func Default() *Context {
	return &Context{
		Ctxt: &build.Default,
	}
}

// Context :
type Context struct {
	Ctxt *build.Context
}

// JoinPath :
func (ctxt *Context) JoinPath(paths ...string) string {
	return buildutil.JoinPath(ctxt.Ctxt, paths...)
}

// IsDir :
func (ctxt *Context) IsDir(path string) bool {
	return buildutil.IsDir(ctxt.Ctxt, path)
}

// SrcDirs :
func (ctxt *Context) SrcDirs() []string {
	return ctxt.Ctxt.SrcDirs()
}

// ReadDir :
func (ctxt *Context) ReadDir(path string) ([]os.FileInfo, error) {
	return buildutil.ReadDir(ctxt.Ctxt, path)
}

// OpenFile :
func (ctxt *Context) OpenFile(path string) (io.ReadCloser, error) {
	return buildutil.OpenFile(ctxt.Ctxt, path)
}

package build

import (
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"golang.org/x/tools/go/buildutil"
)

// Default :
func Default() *Context {
	return &Context{
		Ctxt: &build.Default,
		WriteFile: func(path string, b []byte) error {
			return ioutil.WriteFile(path, b, 0744)
		},
		MkdirAll: func(path string) error {
			return os.MkdirAll(path, 0744)
		},
		MoveFile: func(src, dst string) error {
			if err := os.Chdir(src); err != nil {
				return err
			}
			return exec.Command("git", "mv", src, dst).Run()
		},
	}
}

// Context :
type Context struct {
	Ctxt      *build.Context
	WriteFile func(path string, b []byte) error
	MkdirAll  func(path string) error
	MoveFile  func(src, dst string) error
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

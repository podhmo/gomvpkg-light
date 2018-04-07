package build

import (
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/go/buildutil"
)

// Default :
func Default() *Context {
	var c *Context
	c = &Context{
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
			fs, err := c.ReadDir(src)
			if err != nil {
				return err
			}
			c.MkdirAll(dst)
			for _, f := range fs {
				if strings.HasSuffix(f.Name(), ".go") {
					log.Println("git", "mv", c.JoinPath(src, f.Name()), c.JoinPath(dst, f.Name()))
					if err := exec.Command("git", "mv", c.JoinPath(src, f.Name()), c.JoinPath(dst, f.Name())).Run(); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return c
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

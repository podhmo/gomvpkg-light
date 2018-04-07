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

// Recursively :
func Recursively() *Context {
	var c *Context
	c = &Context{
		Ctxt: &build.Default,
		MatchPkg: func(this, other string) bool {
			return strings.HasPrefix(other, this)
		},
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
	return c
}

// OnePackageOnly :
func OnePackageOnly() *Context {
	var c *Context
	c = &Context{
		Ctxt: &build.Default,
		MatchPkg: func(this, other string) bool {
			return this == other
		},
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

// Default :
var Default = Recursively

// Context :
type Context struct {
	Ctxt      *build.Context
	MatchPkg  func(this, other string) bool
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

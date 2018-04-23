package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/podhmo/gomvpkg-light/build"
	"golang.org/x/tools/go/buildutil"
)

type fakeDirInfo string

func (fd fakeDirInfo) Name() string    { return string(fd) }
func (fakeDirInfo) Sys() interface{}   { return nil }
func (fakeDirInfo) ModTime() time.Time { return time.Time{} }
func (fakeDirInfo) IsDir() bool        { return true }
func (fakeDirInfo) Size() int64        { return 0 }
func (fakeDirInfo) Mode() os.FileMode  { return 0755 }

// Simplifying wrapper around buildutil.FakeContext for packages whose
// filenames are sequentially numbered (%d.go).  pkgs maps a package
// import path to its list of file contents.
func fakeContext(pkgs map[string][]string) *build.OriginalContext {
	pkgs2 := make(map[string]map[string]string)
	for path, files := range pkgs {
		filemap := make(map[string]string)
		for i, contents := range files {
			filemap[fmt.Sprintf("%d.go", i)] = contents
		}
		pkgs2[path] = filemap
	}
	return FakeContext(pkgs2)
}

// FakeContext : replacement for buildutil.FakeContext
func FakeContext(pkgs2 map[string]map[string]string) *build.OriginalContext {
	ctxt := buildutil.FakeContext(pkgs2)

	// fix golang.org/x/tools/go/buildutil/fakecontext.go's readdir
	originalReadDir := ctxt.ReadDir
	clean := func(filename string) string {
		f := path.Clean(filepath.ToSlash(filename))
		// Removing "/go/src" while respecting segment
		// boundaries has this unfortunate corner case:
		if f == "/go/src" {
			return ""
		}
		return strings.TrimPrefix(f, "/go/src/")
	}
	ctxt.ReadDir = func(dir string) ([]os.FileInfo, error) {
		fis, err := originalReadDir(dir)
		pkg := clean(dir)
		for pkgname := range pkgs2 {
			if pkgname == pkg {
				continue
			}
			if strings.HasPrefix(pkgname, pkg+"/") {
				fis = append(fis, fakeDirInfo(strings.TrimPrefix(pkgname, pkg)))
			}
		}
		return fis, err
	}
	return ctxt
}

func TestMoves(t *testing.T) {
	// from: golang.org/x/tools/refactor/rename/mvpkg_test.go
	tests := []struct {
		ctxt         *build.OriginalContext
		from, to, in string
		want         map[string]string
	}{
		// Simple example.
		{
			ctxt: fakeContext(map[string][]string{
				"foo": {`package foo; type T int`},
				"bar": {`package bar`},
				"main": {`package main

import "foo"

var _ foo.T
`},
			}),
			from: "foo", to: "bar", in: "main",
			want: map[string]string{
				"/go/src/main/0.go": `package main

import "bar"

var _ bar.T
`,
				"/go/src/bar/0.go": `package bar

type T int
`,
			},
		},

		// Example with subpackage.
		{
			ctxt: fakeContext(map[string][]string{
				"foo":     {`package foo; type T int`},
				"foo/sub": {`package sub; type T int`},
				"main": {`package main

import "foo"
import "foo/sub"

var _ foo.T
var _ sub.T
`},
			}),
			from: "foo", to: "bar", in: "main",
			want: map[string]string{
				"/go/src/main/0.go": `package main

import "bar"
import "bar/sub"

var _ bar.T
var _ sub.T
`,
				"/go/src/bar/0.go": `package bar

type T int
`,
				"/go/src/bar/sub/0.go": `package sub; type T int`,
			},
		},

		// References into subpackages
		{
			ctxt: fakeContext(map[string][]string{
				"foo":   {`package foo; import "foo/a"; var _ a.T`},
				"foo/a": {`package a; type T int`},
				"foo/b": {`package b; import "foo/a"; var _ a.T`},
			}),
			from: "foo", to: "bar", in: "foo",
			want: map[string]string{
				"/go/src/bar/0.go": `package bar

import "bar/a"

var _ a.T
`,
				"/go/src/bar/a/0.go": `package a; type T int`,
				"/go/src/bar/b/0.go": `package b

import "bar/a"

var _ a.T
`,
			},
		},

		// References into subpackages where directories have overlapped names
		{
			ctxt: fakeContext(map[string][]string{
				"foo":    {},
				"foo/a":  {`package a`},
				"foo/aa": {`package bar`},
				"foo/c":  {`package c; import _ "foo/bar";`},
			}),
			from: "foo/a", to: "foo/spam", in: "foo",
			want: map[string]string{
				"/go/src/foo/spam/0.go": `package spam
`,
				"/go/src/foo/aa/0.go": `package bar`,
				"/go/src/foo/c/0.go":  `package c; import _ "foo/bar";`,
			},
		},

		// External test packages
		{
			ctxt: FakeContext(map[string]map[string]string{
				"foo": {
					"0.go":      `package foo; type T int`,
					"0_test.go": `package foo_test; import "foo"; var _ foo.T`,
				},
				"baz": {
					"0_test.go": `package baz_test; import "foo"; var _ foo.T`,
				},
			}),
			from: "foo", to: "bar", in: "",
			want: map[string]string{
				"/go/src/bar/0.go": `package bar

type T int
`,
				"/go/src/bar/0_test.go": `package bar_test

import "bar"

var _ bar.T
`,
				"/go/src/baz/0_test.go": `package baz_test

import "bar"

var _ bar.T
`,
			},
		},
	}
	for _, test := range tests {
		test := test
		ctxt := build.Recursively()
		ctxt.Ctxt = test.ctxt

		got := make(map[string]string)
		// Populate got with starting file set. rewriteFile and moveDirectory
		// will mutate got to produce resulting file set.
		buildutil.ForEachPackage(test.ctxt, func(importPath string, err error) {
			if err != nil {
				return
			}
			path := filepath.Join("/go/src", importPath, "0.go")
			if !buildutil.FileExists(test.ctxt, path) {
				return
			}
			f, err := test.ctxt.OpenFile(path)
			if err != nil {
				t.Errorf("unexpected error opening file: %s", err)
				return
			}
			bytes, err := ioutil.ReadAll(f)
			f.Close()
			if err != nil {
				t.Errorf("unexpected error reading file: %s", err)
				return
			}
			got[path] = string(bytes)
		})

		ctxt.WriteFile = func(filename string, content []byte) error {
			got[filename] = string(content)
			return nil
		}
		ctxt.MkdirAll = func(path string) error {
			return nil
		}
		ctxt.MoveFile = func(from, to string) error {
			for path, contents := range got {
				if !(strings.HasPrefix(path, from) &&
					(len(path) == len(from) || path[len(from)] == filepath.Separator)) {
					continue
				}
				newPath := strings.Replace(path, from, to, 1)
				delete(got, path)
				got[newPath] = contents
			}
			return nil
		}

		err := run(ctxt, &option{fromPkg: test.from, toPkg: test.to, inPkg: test.in})
		prefix := fmt.Sprintf("-from %q -to %q", test.from, test.to)

		if err != nil {
			t.Errorf("%s: unexpected error: %s", prefix, err)
			continue
		}

		for file, wantContent := range test.want {
			k := filepath.FromSlash(file)
			gotContent, ok := got[k]
			delete(got, k)
			if !ok {
				// TODO(matloob): some testcases might have files that won't be
				// rewritten
				t.Errorf("%s: file %s not rewritten", prefix, file)
				continue
			}
			if gotContent != wantContent {
				t.Errorf("%s: rewritten file %s does not match expectation; got <<<%s>>>\n"+
					"want <<<%s>>>", prefix, file, gotContent, wantContent)
			}
		}

		// got should now be empty
		for file := range got {
			t.Errorf("%s: unexpected rewrite of file %s", prefix, file)
		}
	}
}

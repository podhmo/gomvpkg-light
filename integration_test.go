package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/podhmo/gomvpkg-light/build"
	"golang.org/x/tools/go/buildutil"
)

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
	return buildutil.FakeContext(pkgs2)
}

func TestMoves(t *testing.T) {
	// from: golang.org/x/tools/refactor/rename/mvpkg_test.go
	tests := []struct {
		msg          string
		ctxt         *build.OriginalContext
		from, to, in string
		want         map[string]string
	}{
		{
			msg: "Simple example",
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
		{
			msg: "Example with subpackage",
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
	}
	for _, test := range tests {
		test := test
		t.Run(test.msg, func(t *testing.T) {
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
					if strings.HasPrefix(path, from) {
						newPath := strings.Replace(path, from, to, 1)
						delete(got, path)
						got[newPath] = contents
					}
				}
				return nil
			}

			err := run(ctxt, &option{fromPkg: test.from, toPkg: test.to, inPkg: test.in})
			prefix := fmt.Sprintf("-from %q -to %q", test.from, test.to)

			if err != nil {
				t.Errorf("%s: unexpected error: %s", prefix, err)
				return
			}

			for file, wantContent := range test.want {
				k := filepath.FromSlash(file)
				gotContent, ok := got[k]
				delete(got, k)
				if !ok {
					// TODO(matloob): some testcases might have files that won't be
					// rewritten
					t.Errorf("%s: file %s not rewritten", prefix, file)
					return
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

		})
	}
}

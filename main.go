package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/podhmo/gomvpkg-light/build"
	"github.com/podhmo/gomvpkg-light/collect"
	"github.com/podhmo/gomvpkg-light/move"
	"golang.org/x/tools/go/loader"
)

var (
	fromFlag = flag.String("from", "", "Import path of package to be moved")
	toFlag   = flag.String("to", "", "Destination import path for package")
	inFlag   = flag.String("in", "", "target area")
	helpFlag = flag.Bool("help", false, "show usage message")
)

const Usage = `gomvpkg-light: moves a package, updating import declarations

Usage:

 gomvpkg-light -from <path> -to <path> -in <path>

Flags:

-from        specifies the import path of the package to be moved

-to          specifies the destination import path

-in         specifies the target area of replacing

Examples:

% gomvpkg-light -from myproject/foo -to myproject/bar

  Move the package with import path "myproject/foo" to the new path
  "myproject/bar".
`

func main() {
	flag.Parse()

	if len(flag.Args()) > 0 {
		fmt.Fprintln(os.Stderr, "gomvpkg-light: surplus arguments.")
		os.Exit(1)
	}

	if *helpFlag || *fromFlag == "" || *toFlag == "" || *inFlag == "" {
		fmt.Println(Usage)
		return
	}

	ctxt := build.Default()

	if err := run(ctxt, *fromFlag, *toFlag, *inFlag); err != nil {
		fmt.Fprintf(os.Stderr, "gomvpkg-light: %+v.\n", err)
		os.Exit(1)
	}
}

func run(ctxt *build.Context, fromPkg, toPkg, inPkg string) error {
	st := time.Now()
	defer fmt.Fprintln(os.Stderr, time.Now().Sub(st))

	root, err := collect.TargetRoot(ctxt, inPkg)
	if err != nil {
		return err
	}

	pkgdirs, err := collect.GoFilesDirectories(ctxt, root)
	if err != nil {
		return err
	}

	affected, err := collect.AffectedPackages(ctxt, fromPkg, root, pkgdirs)
	if err != nil {
		return err
	}

	// slow
	c := loader.Config{
		TypeCheckFuncBodies: func(path string) bool {
			if strings.Contains(path, "/vendor/") {
				return false
			}
			return true
		},
	}

	for _, a := range affected {
		c.Import(a.Pkg)
	}

	prog, err := c.Load()
	if err != nil {
		return err
	}
	log.Println(len(prog.AllPackages))

	req := &move.Req{
		FromPkg:  fromPkg,
		ToPkg:    toPkg,
		InPkg:    inPkg,
		Root:     root,
		Affected: affected,
	}

	// todo: check
	if err := move.TargetPackage(prog, req); err != nil {
		return err
	}

	if err := move.AffectedPackages(prog, req); err != nil {
		return err
	}

	fmt.Println("ok")
	return nil
}

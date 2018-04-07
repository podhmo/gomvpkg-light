package main

import (
	"fmt"
	"go/token"
	"log"
	"os"
	"strings"
	"time"

	"github.com/podhmo/gomvpkg-light/build"
	"github.com/podhmo/gomvpkg-light/collect"
	"github.com/podhmo/gomvpkg-light/move"
	"golang.org/x/tools/go/loader"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type option struct {
	fromPkg string
	toPkg   string
	inPkg   string
}

func main() {
	var option option
	cmd := kingpin.New("gomvpkg-light", "gomvpkg-light")

	cmd.Flag("from", "Import path of package to be moved").Required().StringVar(&option.fromPkg)
	cmd.Flag("to", "Destination import path for package").StringVar(&option.toPkg)
	cmd.Flag("in", "target area").StringVar(&option.inPkg)

	if _, err := cmd.Parse(os.Args[1:]); err != nil {
		cmd.FatalUsage(err.Error())
	}

	ctxt := build.Default()

	if err := run(ctxt, &option); err != nil {
		fmt.Fprintf(os.Stderr, "gomvpkg-light: %+v.\n", err)
		os.Exit(1)
	}
}

func run(ctxt *build.Context, option *option) error {
	st := time.Now()
	defer fmt.Fprintln(os.Stderr, time.Now().Sub(st))

	root, err := collect.TargetRoot(ctxt, option.inPkg)
	if err != nil {
		return err
	}

	pkgdirs, err := collect.GoFilesDirectories(ctxt, root)
	if err != nil {
		return err
	}

	affected, err := collect.AffectedPackages(ctxt, option.fromPkg, root, pkgdirs)
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
		c.ImportWithTests(a.Pkg)
	}

	prog, err := c.Load()
	if err != nil {
		return err
	}
	log.Println(len(prog.AllPackages))

	req := &move.Req{
		FromPkg:     option.fromPkg,
		ToPkg:       option.toPkg,
		InPkg:       option.inPkg,
		Root:        root,
		Affected:    affected,
		WillBeWrite: map[*token.File]*move.PreWrite{},
	}

	// todo: check
	if err := move.TargetPackage(prog, req); err != nil {
		return err
	}

	if err := move.AffectedPackages(prog, req); err != nil {
		return err
	}

	for f := range req.WillBeWrite {
		fmt.Println(f.Name())
	}
	fmt.Println("ok")
	return nil
}

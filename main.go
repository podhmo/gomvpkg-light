package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
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
		log.Fatalf("gomvpkg-light: %+v.\n", err)
	}
}

func run(ctxt *build.Context, option *option) error {
	log.Printf("start move package %s -> %s", option.fromPkg, option.toPkg)
	st := time.Now()
	defer func() {
		log.Printf("takes %v", time.Now().Sub(st))
		log.Println("end")
	}()

	root, err := collect.TargetRoot(ctxt, option.inPkg)
	if err != nil {
		return err
	}
	log.Printf("get in-pkg %s", root.Path)

	pkgdirs, err := collect.GoFilesDirectories(ctxt, root)
	if err != nil {
		return err
	}
	log.Printf("collect candidate directories %d", len(pkgdirs))

	affected, err := collect.AffectedPackages(ctxt, option.fromPkg, root, pkgdirs)
	if err != nil {
		return err
	}
	log.Printf("collect affected packages %d", len(affected))

	// slow
	c := loader.Config{
		TypeCheckFuncBodies: func(path string) bool {
			if strings.Contains(path, "/vendor/") {
				return false
			}
			return true
		},
		ParserMode: parser.ParseComments,
	}

	c.ImportWithTests(option.fromPkg)
	for _, a := range affected {
		c.ImportWithTests(a.Pkg)
	}

	prog, err := c.Load()
	if err != nil {
		return err
	}

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

	srctarget, err := collect.TargetRoot(ctxt, option.fromPkg)
	if err != nil {
		return errors.Errorf("invalid source %s", option.fromPkg)
	}
	dsttarget, err := collect.TargetRoot(ctxt, option.toPkg)
	if err != nil {
		dsttarget = &collect.Target{
			Dir:        srctarget.Dir,
			Pkg:        option.toPkg,
			Path:       ctxt.JoinPath(srctarget.Dir, option.toPkg),
			NeedCreate: true,
		}
	}

	pp := &printer.Config{Tabwidth: 8, Mode: printer.UseSpaces | printer.TabIndent}

	for f, pw := range req.WillBeWrite {
		var b bytes.Buffer
		if err := pp.Fprint(&b, prog.Fset, pw.File); err != nil {
			return err
		}

		if err := ctxt.WriteFile(f.Name(), b.Bytes()); err != nil {
			return err
		}
		log.Printf("write file %s", f.Name())
	}

	if dsttarget.NeedCreate {
		if err := ctxt.MkdirAll(filepath.Dir(dsttarget.Path)); err != nil {
			return err
		}
	}

	log.Printf("move package %s -> %s", srctarget.Pkg, dsttarget.Pkg)
	if err := ctxt.MoveFile(srctarget.Path, dsttarget.Path); err != nil {
		return err
	}

	// debug
	// for f, pw := range req.WillBeWrite {
	// 	fmt.Println("----------------------------------------")
	// 	fmt.Println(f.Name())
	// 	fmt.Println("----------------------------------------")
	// 	pp.Fprint(os.Stdout, prog.Fset, pw.File)
	// }

	fmt.Println("ok")
	return nil
}

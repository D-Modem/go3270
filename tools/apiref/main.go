// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2026 D-Modem Authors

// SPDX-License-Identifier: MIT
//
// apiref writes docs/api.md, the API reference of the site, from the
// package's own doc comments (go/doc, as go doc and pkg.go.dev read them).
// It is a module of its own so that go3270's go.mod stays at Go 1.18.
//
//	cd tools/apiref && go run . ../.. ../../docs/api.md
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: apiref <package dir> <out.md>")
		os.Exit(2)
	}
	out, err := reference(os.Args[1])
	if err == nil {
		err = os.WriteFile(os.Args[2], out, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "apiref:", err)
		os.Exit(1)
	}
}

func reference(dir string) ([]byte, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var files []*ast.File
	name := ""
	for n, p := range pkgs {
		name = n
		for _, f := range p.Files {
			files = append(files, f)
		}
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("%s: want one package, have %d", dir, len(pkgs))
	}
	pkg, err := doc.NewFromFiles(fset, files, "github.com/D-Modem/go3270")
	if err != nil {
		return nil, err
	}

	r := &writer{fset: fset, pkg: pkg}
	r.printf("<!-- Made by tools/apiref from the package's doc comments: change the code, not this page. -->\n\n")
	r.printf("# API reference\n\n")
	r.printf("The exported API of package `%s`, from its doc comments (the same text "+
		"that `go doc -all` and [pkg.go.dev](https://pkg.go.dev/github.com/D-Modem/go3270) show).\n\n", name)
	r.printf("    import \"%s\"\n\n", pkg.ImportPath)
	r.text(pkg.Doc)

	r.values("Constants", pkg.Consts)
	r.values("Variables", pkg.Vars)
	if len(pkg.Funcs) > 0 {
		r.printf("## Functions\n\n")
		for _, f := range pkg.Funcs {
			r.fn(f, "###")
		}
	}
	if len(pkg.Types) > 0 {
		r.printf("## Types\n\n")
		for _, t := range pkg.Types {
			r.printf("### %s\n\n", t.Name)
			r.code(t.Decl)
			r.text(t.Doc)
			for _, v := range append(t.Consts, t.Vars...) {
				r.code(v.Decl)
				r.text(v.Doc)
			}
			for _, f := range t.Funcs {
				r.fn(f, "####")
			}
			for _, f := range t.Methods {
				r.fn(f, "####")
			}
		}
	}
	return bytes.TrimRight(r.b.Bytes(), "\n"), nil
}

type writer struct {
	b    bytes.Buffer
	fset *token.FileSet
	pkg  *doc.Package
}

func (r *writer) printf(f string, a ...any) { fmt.Fprintf(&r.b, f, a...) }

func (r *writer) code(n any) {
	var b bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	if err := cfg.Fprint(&b, r.fset, n); err != nil {
		panic(err)
	}
	r.printf("```go\n%s\n```\n\n", b.String())
}

// text is a doc comment as Markdown, its headings below the page's.
func (r *writer) text(s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	p := r.pkg.Printer()
	p.HeadingLevel = 4
	p.DocLinkURL = func(l *comment.DocLink) string {
		if l.ImportPath == "" || l.ImportPath == r.pkg.ImportPath {
			name := l.Name
			if l.Recv != "" {
				name = l.Recv + "." + l.Name
			}
			return "#" + strings.ToLower(strings.ReplaceAll(name, ".", ""))
		}
		return l.DefaultURL("https://pkg.go.dev")
	}
	r.b.Write(p.Markdown(r.pkg.Parser().Parse(s)))
	r.printf("\n")
}

func (r *writer) values(title string, vs []*doc.Value) {
	if len(vs) == 0 {
		return
	}
	r.printf("## %s\n\n", title)
	for _, v := range vs {
		r.code(v.Decl)
		r.text(v.Doc)
	}
}

func (r *writer) fn(f *doc.Func, level string) {
	name := f.Name
	if f.Recv != "" {
		name = strings.TrimPrefix(f.Recv, "*") + "." + f.Name
	}
	r.printf("%s %s\n\n", level, name)
	r.code(f.Decl)
	r.text(f.Doc)
}

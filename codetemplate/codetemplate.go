/*
Package codetemplate uses a fork of "text/template" to print Go code more concisely.

Users of this package can write "text/template"-style templates for Go code, and
any *codegenutil.Symbol that appears as a template variable will be formatting
using the GoCode() function using the *codegenutil.FileImports of the file being
generated. Furthermore, {{header}} and {{imports}} may be placed in the template
text to output "package foo \n imports(...)" or "imports(...)" respectively.

    this is an exaple

The "text/template" package has some limitations that make it cumbersome to use
for printing Go code. Namely, "fmt.Fprint" is used to print values to the output
stream. The fmt package makes it impossible to pass extra contextual information
to the object being printed. Because of this limitation, the codetemplate
package uses a fork of the "text/template" package that allows using something
other than fmt.Fprint to format objects.
*/
package codetemplate

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/meta-programming/go-codegenutil"
	"github.com/meta-programming/go-codegenutil/template"
	"github.com/meta-programming/go-codegenutil/unusedimports"
)

// Option parameterizes Template construction.
type Option struct {
	apply func(t *Template)
}

func KeepUnusedImports() Option {
	return Option{func(t *Template) { t.formatter = nil }}
}

// WithName specifies the name of the text template creates.
func WithName(templateName string) Option {
	return Option{func(t *Template) { t.templateName = templateName }}
}

// WithFuncs specifies the name of the text template creates.
func WithFuncs(funcs template.FuncMap) Option {
	return Option{func(t *Template) {
		t.transformers = append(t.transformers, func(tmpl *template.Template) {
			tmpl.Funcs(funcs)
		})
	}}
}

// Template is a Go code generation template. See Parse() for details.
type Template struct {
	tt                 *template.Template
	importsPlaceholder string
	headerPlaceholder  string

	templateName string
	// called in successon on the template during construction
	transformers []func(tmpl *template.Template)
	formatter    func(filename, code string) (string, error)
}

// Parse returns a new template by passing tmplText to the parser in
// "text/template".
//
// The template is evaluated with additional "pipeline" functions:
//
//    imports
//             A function that takes no arguments and outputs an imports
//             block, a.k.a. ImportDecl in the Go spec:
//             https://go.dev/ref/spec#ImportDecl.
//    header
//             A function that takes no arguments and outputs a package statement
//             and imports block, a.ka. PackageClause and ImportDecl in the Go spec:
//             https://go.dev/ref/spec#SourceFile.
func Parse(tmplText string, opts ...Option) (*Template, error) {
	h := sha256.New()
	h.Write([]byte(tmplText))
	importsPlaceholder := fmt.Sprintf("<PLACEHOLDER FOR IMPORTS %x>", h.Sum(nil))
	headerPlaceholder := fmt.Sprintf("<PLACEHOLDER FOR PACKAGE STATEMENT AND IMPORTS %x>", h.Sum(nil))

	out := &Template{
		importsPlaceholder: importsPlaceholder,
		headerPlaceholder:  headerPlaceholder,
		formatter:          unusedimports.PruneUnparsed,
		templateName:       "generated.go",
	}
	for _, opt := range opts {
		opt.apply(out)
	}

	t := template.New(out.templateName)
	for _, transformer := range out.transformers {
		transformer(t)
	}

	t, err := t.Funcs(template.FuncMap{
		"imports": func() string {
			return importsPlaceholder
		},
		"header": func() string {
			return headerPlaceholder
		},
	}).Option("missingkey=error").Parse(tmplText)
	if err != nil {
		return nil, err
	}
	out.tt = t
	return out, nil
}

func (t *Template) Execute(imports *codegenutil.FileImports, wr io.Writer, data any) error {
	execT, err := t.tt.Clone()
	if err != nil {
		return fmt.Errorf("error with Clone: %w", err)
	}
	execT.Printer(false, t.makePrinter(imports))

	pass1Buf := &strings.Builder{}
	// Pass 1
	if err := execT.Execute(pass1Buf, data); err != nil {
		return err
	}

	withImports := strings.ReplaceAll(pass1Buf.String(), t.importsPlaceholder, imports.Format(false))
	withHeader := strings.ReplaceAll(withImports, t.headerPlaceholder, imports.Format(true))

	formatted, err := withHeader, error(nil)
	if t.formatter != nil {
		formatted, err = t.formatter("", withHeader)
	}
	if err != nil {
		return fmt.Errorf("error formatting template output: %w", err)
	}

	if _, err := wr.Write([]byte(formatted)); err != nil {
		return err
	}

	return nil
}

func (t *Template) makePrinter(imports *codegenutil.FileImports) template.FormatFunc {
	// TODO: Add an option to NewTemplate that allows customizing this function.
	return func(w io.Writer, raw any) (n int, err error) {
		outStr := ""
		switch obj := raw.(type) {
		case interface {
			GoCode(*codegenutil.FileImports) string
		}:
			outStr = obj.GoCode(imports)
		default:
			outStr = fmt.Sprint(raw)
		}

		return w.Write([]byte(outStr))
	}
}

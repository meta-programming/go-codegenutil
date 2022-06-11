package codetemplate

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/meta-programming/go-codegenutil"
	"github.com/meta-programming/go-codegenutil/template"
)

type Template struct {
	tt                 *template.Template
	importsPlaceholder string
	headerPlaceholder  string
}

func Parse(name, tmplText string) (*Template, error) {
	h := sha256.New()
	h.Write([]byte(tmplText))
	importsPlaceholder := fmt.Sprintf("<PLACEHOLDER FOR IMPORTS %x>", h.Sum(nil))
	headerPlaceholder := fmt.Sprintf("<PLACEHOLDER FOR PACKAGE STATEMENT AND IMPORTS %x>", h.Sum(nil))

	out := &Template{
		importsPlaceholder: importsPlaceholder,
		headerPlaceholder:  headerPlaceholder,
	}
	t, err := template.New(name).Funcs(template.FuncMap{
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
	execT.Printer(func(w io.Writer, a ...any) (n int, err error) {
		for _, raw := range a {
			outStr := ""
			switch obj := raw.(type) {
			case codegenutil.Symbol:
				outStr = obj.GoCode(imports)
			case interface {
				GoCode(*codegenutil.FileImports) string
			}:
				outStr = obj.GoCode(imports)
			default:
				outStr = fmt.Sprint(raw)
			}

			nn, err := w.Write([]byte(outStr))
			n += nn
			if err != nil {
				return n, err
			}
		}
		return n, nil
	})

	pass1Buf := &strings.Builder{}
	// Pass 1
	if err := execT.Execute(pass1Buf, data); err != nil {
		return err
	}

	withImports := strings.ReplaceAll(pass1Buf.String(), t.importsPlaceholder, imports.Format(false))
	withHeader := strings.ReplaceAll(withImports, t.headerPlaceholder, imports.Format(true))

	if _, err := wr.Write([]byte(withHeader)); err != nil {
		return err
	}

	return nil
}

package codetemplate

import (
	"bytes"
	"testing"

	"github.com/meta-programming/go-codegenutil"
	"github.com/meta-programming/go-codegenutil/debugutil"
)

func TestTemplate_Execute(t *testing.T) {
	pkg1 := codegenutil.AssumedPackageName("abc.xyz/mypkg")
	tests := []struct {
		name         string
		template     string
		imports      *codegenutil.FileImports
		data         any
		want         string
		wantErr      bool
		wantParseErr bool
	}{
		{
			name: "example1",
			template: `// docs
{{header}}

// Doesn't do anything special, really.
var myThing = {{.mysym}}
var myThing2 = {{.mysym2}}

const myNum {{.numType}} = 42
`,
			imports: codegenutil.NewFileImports(pkg1),
			data: map[string]*codegenutil.Symbol{
				"mysym":   codegenutil.AssumedPackageName("math").Symbol("Max"),
				"mysym2":  codegenutil.AssumedPackageName("alternative/math").Symbol("Max"),
				"numType": codegenutil.Sym("", "int64"),
			},
			want: `// docs
package mypkg

import (
	"math"

	math2 "alternative/math"
)

// Doesn't do anything special, really.
var myThing = math.Max
var myThing2 = math2.Max

const myNum int64 = 42
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Parse(tt.template)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("Parse got error %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			wr := &bytes.Buffer{}
			if err := tmpl.Execute(tt.imports, wr, tt.data); (err != nil) != tt.wantErr {
				t.Errorf("Template.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWr := wr.String(); gotWr != tt.want {
				t.Errorf("Template.Execute() generated unexpected output (want|got):\n%s", debugutil.SideBySide(gotWr, tt.want))
			}
		})
	}
}

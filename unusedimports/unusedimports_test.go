// Package unusedimports provides the import pruning features of the `goimports`
// command in library form.
package unusedimports

import (
	"testing"

	"github.com/meta-programming/go-codegenutil/debugutil"
)

func TestPruneUnparsed(t *testing.T) {
	tests := []struct {
		filename string
		src      string
		want     string
		wantErr  bool
	}{
		{
			filename: "simple.go",
			src: `package foo
import "bar"
`,
			want: `package foo
`,
		},
		{
			filename: "renamed.go",
			src: `package foo
import x "bar"
`,
			want: `package foo
`,
		},
		{
			filename: "needspruning.go",
			src: `package foo
import x "bar"

func foo() {

}
`,
			want: `package foo

func foo() {

}
`,
		},
		{
			filename: "noprune.go",
			src: `package foo

import x "bar"

func foo() {
	x.Boom()
}
`,
			want: `package foo

import x "bar"

func foo() {
	x.Boom()
}
`,
		},
		{
			filename: "blank.go",
			src: `package foo

import _ "bar"

func foo() {
	x.Boom()
}
`,
			want: `package foo

import _ "bar"

func foo() {
	x.Boom()
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got, err := PruneUnparsed(tt.filename, tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("PruneUnparsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PruneUnparsed() generated unexpected output (want|got):\n%s", debugutil.SideBySide(tt.want, got))
			}
		})
	}
}

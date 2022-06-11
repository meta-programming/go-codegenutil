// Package unusedimports provides the import pruning features of the `goimports`
// command in library form.
package unusedimports

import (
	"fmt"
	"strings"
	"testing"
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
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got, err := PruneUnparsed(tt.filename, tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("PruneUnparsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PruneUnparsed() generated unexpected output (want|got):\n%s", sideBySide(tt.want, got))
			}
		})
	}
}

func sideBySide(a, b string) string {
	linesA := strings.Split(replaceTabs(a), "\n")
	linesB := strings.Split(replaceTabs(b), "\n")
	lhsWidth := maxWidth(linesA)

	lineOrBlank := func(lines []string, i int) string {
		if i < len(lines) {
			return lines[i]
		}
		return ""
	}

	var outLines []string
	for i := 0; ; i++ {
		if i >= len(linesA) && i >= len(linesB) {
			break
		}

		lineA := lineOrBlank(linesA, i)
		lineB := lineOrBlank(linesB, i)
		outLines = append(outLines, fmt.Sprintf("%s|%s", pad(lineA, lhsWidth), lineB))
	}
	return strings.Join(outLines, "\n")
}

func replaceTabs(str string) string {
	return strings.ReplaceAll(str, "\t", "  ")
}

func maxWidth(lines []string) int {
	max := 0
	for _, line := range lines {
		l := len(line)
		if l > max {
			max = l
		}
	}
	return max
}

func pad(line string, width int) string {
	out := line
	padSize := width - len(line)
	for i := 0; i < padSize; i++ {
		out += " "
	}
	return out
}

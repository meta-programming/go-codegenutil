package codegenutil

import (
	"testing"
)

func TestAssumedPackageName(t *testing.T) {
	tests := []struct {
		importPath string
		want       *Package
	}{
		{
			importPath: "go.lang/x/tools",
			want:       &Package{importPath: "go.lang/x/tools", name: "tools"},
		},
		{
			importPath: "go.lang/x/tools/v2",
			want:       &Package{importPath: "go.lang/x/tools/v2", name: "tools"},
		},
		{
			importPath: "go.lang/x/go-tools/v2",
			want:       &Package{importPath: "go.lang/x/go-tools/v2", name: "tools"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			if got := AssumedPackageName(tt.importPath); !pkgEqual(got, tt.want) {
				t.Errorf("AssumedPackageName() = %q, want %v", got, tt.want)
			}
		})
	}
}

func pkgEqual(a, b *Package) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.importPath == b.importPath && a.name == b.name
}

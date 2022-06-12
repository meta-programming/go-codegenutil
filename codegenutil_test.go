package codegenutil

import (
	"testing"
)

func TestIdentifierRegexp(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"helloWorld123", true},
		{"_helloWorld123", true},
		{"a", true},
		{"_ó3", true},
		{"b_b", true},
		{"A_b", true},
		{"A b", false},
		{"A-b", false},
	}
	for _, tt := range tests {
		if got, want := identifierRegexp.MatchString(tt.id), tt.want; got != want {
			t.Errorf("%q got is regexp = %v, want = %v", tt.id, got, want)
		}
	}
}

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

func TestSymbol_GoCode(t *testing.T) {
	type example struct {
		name    string
		sym     *Symbol
		imports *FileImports
		want    string
	}
	tests := []example{
		{
			name:    "simple1",
			sym:     AssumedPackageName("abc/xyz").Symbol("Foo"),
			imports: NewFileImports(AssumedPackageName("bar")),
			want:    "xyz.Foo",
		},
		{
			name:    "symbol in package of file",
			sym:     AssumedPackageName("abc/xyz").Symbol("Foo"),
			imports: NewFileImports(AssumedPackageName("abc/xyz")),
			want:    "Foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sym.GoCode(tt.imports); got != tt.want {
				t.Errorf("%v.GoCode() = %q, want %q", tt.sym, got, tt.want)
			}
		})
	}
}

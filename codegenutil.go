// Package codegenutil provides utilities for generating code in Go.
//
// The current ontology is as follows:
//
// Package: Uniquely identifies a package using an import path. Also contains
// the "package name," which is typically inferred from the import path.
//
// Imports: Keeps track of a set of imports within a Go file.
//
// Symbol: A (Package, string) pair where the string is an identifer. "math.Max"
// is a textual representation of a symbol where "math" is the Package and "Max"
// is the local identifier.
//
//
package codegenutil

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Package identifies a package by its import path and the package name used to
// declare the package (i.e. the "xyz" in "package xyz" statement).
type Package struct {
	importPath, name string
}

// ImportPath returns the value in the import statement used to import the
// package.
//
// This path typically identifies a package uniquely, but that is not necesarily
// the case from the Go spec. https://go.dev/ref/spec#ImportSpec.
func (p *Package) ImportPath() string { return p.importPath }

// Name is the string that appears in the "package clause" of the defining go
// file.
//
// See https://go.dev/ref/spec#PackageClause and
// https://go.dev/ref/spec#PackageName.
func (p *Package) Name() string { return p.name }

// Symbol returns a new Symbol within the given package.
func (p *Package) Symbol(idName string) *Symbol {
	return &Symbol{p, idName}
}

// FileImports captures information about import entries in a Go file and the
// package of the Go file itself.
type FileImports struct {
	filePackage *Package
	specs       []*ImportSpec

	byLocalPackageName map[string]*ImportSpec
	byImportPath       map[string]*ImportSpec

	// suggestPackageNames is a function that suggests a package name for
	// an import path.
	suggestPackageNames func(pkg *Package, tryImportSpec func(localPackageName string) (acceptable bool))

	rwMutex *sync.RWMutex
}

// FileImportsOption is an option that can be passed to NewImportsFor to customize
// its behavior.
type FileImportsOption struct {
	apply func(*FileImports)
}

// CustomPackageNameSuggester returns an option for customizing how an Imports
// object chooses the package name to use for an import path and whether that
// package name is an alias.
//
// The suggestion function takes three arguments:
//
// 1) importPath is the path of the package being imported, such as "x/y/z" in
// the import `import x "x/y/z"`.
//
// 2) packageNameInPackageClause is the name of the package from the package
// clause of Go files that define the package. This is often the last element of
// the import path, but it frequently differs. For example, "blah" is typically
// the package name for an import like "x/y/blah/v3" because of how Go's module
// system works. This value may also be the empty string, which indicates the
// package name is unknown.
//
// 3) callback is the function that should be called with package name
// suggestions. fn should call the callback function with different candidate
// names until the callback returns false, at which point the function should
// stop suggesting package names. The arguments to the callback are the package
// name to use and whether or not that package name should be considered an
// alias.
func CustomPackageNameSuggester(fn func(pkg *Package, tryImportSpec func(localPackageName string) (acceptable bool))) FileImportsOption {
	return FileImportsOption{
		func(i *FileImports) { i.suggestPackageNames = fn },
	}
}

// WithImports returns an option that add all of the provided package to the
// returned *FileImports.
func WithImports(pkgs ...*Package) FileImportsOption {
	return FileImportsOption{
		func(fi *FileImports) {
			for _, x := range pkgs {
				fi.Add(x, "")
			}
		},
	}
}

// NewFileImports returns a new *FileImports object with no imports.
func NewFileImports(p *Package, opts ...FileImportsOption) *FileImports {
	fi := &FileImports{
		p,
		nil,
		map[string]*ImportSpec{},
		map[string]*ImportSpec{},
		nil,
		&sync.RWMutex{},
	}
	for _, x := range opts {
		x.apply(fi)
	}
	return fi
}

// Package returns the Package of the file in which the imports appear.
func (fi *FileImports) Package() *Package { return fi.filePackage }

// Find returns the import spec corresponding a given package or nil if the
// package wasn't found.
//
// It is possible to have multiple imports of a package, and this function will
// return the first.
func (fi *FileImports) Find(p *Package) *ImportSpec {
	fi.rwMutex.RLock()
	defer fi.rwMutex.RUnlock()
	return fi.byImportPath[p.ImportPath()]
}

// Add adds an import to the given package using the given alias.
//
// If alias is empty, the import spec will have no alias, and the package name
// of the package will be used.
//
// If the package name or alias conflicts with an existing import, an alias will
// be generated.
func (fi *FileImports) Add(pkg *Package, alias string) *ImportSpec {
	fi.rwMutex.Lock()
	defer fi.rwMutex.Unlock()

	existingSpec := fi.byImportPath[pkg.ImportPath()]
	if existingSpec != nil {
		return existingSpec
	}

	suggester := fi.suggestPackageNames
	if suggester == nil {
		suggester = defaultSuggestPackageNames
	}
	var finalSpec *ImportSpec
	suggester(pkg, func(suggestedPackageName string) (acceptable bool) {
		if _, conflicts := fi.byLocalPackageName[suggestedPackageName]; conflicts {
			return false // keep sugesting
		}
		isExplicit := suggestedPackageName != pkg.Name()
		finalSpec = &ImportSpec{suggestedPackageName, pkg, isExplicit}
		fi.byLocalPackageName[suggestedPackageName] = finalSpec
		fi.byImportPath[pkg.ImportPath()] = finalSpec
		fi.specs = append(fi.specs, finalSpec)
		return true // finished with suggestions
	})
	if finalSpec == nil {
		panic(fmt.Errorf("no acceptable suggestion found for importing %q", pkg.ImportPath()))
	}
	return finalSpec
}

// List returns all of the import specs for the FileImports object.
func (fi *FileImports) List() []*ImportSpec {
	fi.rwMutex.RLock()
	var out []*ImportSpec
	out = append(out, fi.specs...)
	fi.rwMutex.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].PackageName().ImportPath() < out[j].PackageName().ImportPath()
	})
	return out
}

// String prints a valid Go imports block containing all of the imports.
func (fi *FileImports) String() string {
	imports := fi.List()
	var aliasedLines, simpleLines, blankLines []string

	for _, impt := range imports {
		if impt.IsExplicit() && impt.FileLocalPackageName() == "_" {
			blankLines = append(blankLines, fmt.Sprintf("\t%s %q", impt.FileLocalPackageName(), impt.PackageName().ImportPath()))
		} else if impt.IsExplicit() {
			aliasedLines = append(aliasedLines, fmt.Sprintf("\t%s %q", impt.FileLocalPackageName(), impt.PackageName().ImportPath()))
		} else {
			simpleLines = append(simpleLines, fmt.Sprintf("\t%q", impt.PackageName().ImportPath()))
		}
	}
	sections := []string{}
	addSection := func(lines []string) {
		if len(lines) == 0 {
			return
		}
		sections = append(sections, strings.Join(lines, "\n")+"\n")
		if len(sections) == 1 {
			sections[0] = "\n" + sections[0]
		}
	}
	addSection(simpleLines)
	addSection(aliasedLines)
	addSection(blankLines)

	return fmt.Sprintf("import (%s)", strings.Join(sections, "\n"))
}

func prefixLines(lines []string, prefix string) []string {
	var out []string
	for _, line := range lines {
		out = append(out, "\t"+line)
	}
	return out
}

// ImportSpec is an entry within the set of imports of a Go file. It does not
// contain formatting information, like import order.
type ImportSpec struct {
	fileLocalPackageName string
	pkg                  *Package
	isExplicit           bool
}

// FileLocalPackageName returns the package name used to identify this package
// within the Go file using this import spec.
func (is *ImportSpec) FileLocalPackageName() string { return is.fileLocalPackageName }

// PackageName returns the package designated by the import.
func (is *ImportSpec) PackageName() *Package { return is.pkg }

// ImportSpec returns true if the import spec has an explicit package name
// (sometimes called an "alias" or "package name alias")
func (is *ImportSpec) IsExplicit() bool { return is.isExplicit }

// Symbol in this package is used for a (PackageName, string) pair that
// pair
type Symbol struct {
	pkg  *Package
	name string
}

// Package returns the package name of the symbol.
//
// This should not be nil. If symbol is a local symbol for code in a file inside
// package "abc/xyz", the package should be set to
// AssumedPackageName("abc/xzy").
func (s *Symbol) Package() *Package { return s.pkg }

// Name returns the local name of the symbol. For symbol
// "github.com/meta-programming/go-codegenutil".Foo, Name() would return "Foo".
//
// If the symbol is a "QualifiedSymbol"[1], this is the "identifier" part of the
// symbol
func (s *Symbol) Name() string { return s.name }

// FormatEnsureImported formats the symbol in a given printing context.
//
// The Imports argument is the set of imports currently imported in the file. If
// the symbol's import is not in the set of import specs.
func (s *Symbol) FormatEnsureImported(imports *FileImports) string {
	if s.Package().ImportPath() == imports.filePackage.ImportPath() {
		return s.Name()
	}
	return imports.Add(s.Package(), "").FileLocalPackageName() + "." + s.Name()
}

// AssumedPackageName returns the assumed name of the package according the
// the package definition's package clause based purely on the package's import
// path.
//
// Per https://golang.org/ref/spec#Import_declarations: "If the PackageName is
// omitted, it defaults to the identifier specified in the package clause of the
// imported package." The file being loaded is not available in gopoet (and many
// go tools), so this function needs to be used.
//
// Note: path.Base differs from the package name guesser used by most
// tools. See https://pkg.go.dev/golang.org/x/tools/internal/imports#ImportPathToAssumedName.
func AssumedPackageName(importPath string) *Package {
	// Contents of this function are taken from
	// https://pkg.go.dev/golang.org/x/tools@v0.1.10/internal/imports#ImportPathToAssumedName,
	// which has the following license:
	//
	// Copyright (c) 2009 The Go Authors. All rights reserved.
	//
	// Redistribution and use in source and binary forms, with or without
	// modification, are permitted provided that the following conditions are
	// met:
	//
	//    * Redistributions of source code must retain the above copyright
	// notice, this list of conditions and the following disclaimer.
	//    * Redistributions in binary form must reproduce the above
	// copyright notice, this list of conditions and the following disclaimer in
	// the documentation and/or other materials provided with the distribution.
	//    * Neither the name of Google Inc. nor the names of its
	// contributors may be used to endorse or promote products derived from this
	// software without specific prior written permission.
	//
	// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS
	// IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
	// THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
	// PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR
	// CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
	// EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
	// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
	// PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
	// LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
	// NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
	// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

	// notIdentifier reports whether ch is an invalid identifier character.
	notIdentifier := func(ch rune) bool {
		return !('a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' ||
			'0' <= ch && ch <= '9' ||
			ch == '_' ||
			ch >= utf8.RuneSelf && (unicode.IsLetter(ch) || unicode.IsDigit(ch)))
	}
	base := path.Base(importPath)
	if strings.HasPrefix(base, "v") {
		if _, err := strconv.Atoi(base[1:]); err == nil {
			dir := path.Dir(importPath)
			if dir != "." {
				base = path.Base(dir)
			}
		}
	}
	base = strings.TrimPrefix(base, "go-")
	if i := strings.IndexFunc(base, notIdentifier); i >= 0 {
		base = base[:i]
	}
	return &Package{importPath, base}
}

// ExplicitPackageName is used to construct an explicit PackageName in case
// AssumedPackageName is insufficient.
func ExplicitPackageName(importPath, packageName string) *Package {
	return &Package{importPath, packageName}
}

// defaultSuggestPackageNames calls callback with a series of suggested package names
// for the given importPath and assumed package name until the callback returns
// false.
func defaultSuggestPackageNames(pkg *Package, tryImportSpec func(localPackageName string) (accepted bool)) {
	packageNameInPackageClause := pkg.Name()

	if tryImportSpec(packageNameInPackageClause) {
		return
	}

	const maxIterations = 1000
	for suffix := 1; suffix <= maxIterations; suffix++ {
		packageName := fmt.Sprintf("%s%d", packageNameInPackageClause, suffix)
		if tryImportSpec(packageName) {
			return
		}
	}
	// Nothing accepted - give up without panic.
}

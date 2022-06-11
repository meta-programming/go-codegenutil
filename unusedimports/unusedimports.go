// Package unusedimports provides the import pruning features of the `goimports`
// command in library form. Intended usage is for code generator libraries.
package unusedimports

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"github.com/meta-programming/go-codegenutil"
	"golang.org/x/tools/go/ast/astutil"
)

// PruneUnparsed parses a Go file and removes unused imports.
//
// The filename argument is used only for printing error messages.
func PruneUnparsed(filename, src string) (string, error) {
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	if err := pruneAlreadyParsed(fset, f); err != nil {
		return "", err
	}

	// Print the AST.
	out := &strings.Builder{}
	printer.Fprint(out, fset, f)
	return out.String(), nil
}

// pruneAlreadyParsed modifies fset by removing unused imports.
func pruneAlreadyParsed(fset *token.FileSet, file *ast.File) error {

	refs := collectReferences(file)
	imports := collectImports(file)
	p := &pass{}

	existingImports := make(map[string]*importInfo)
	for _, imp := range imports {
		existingImports[p.importIdentifier(imp)] = imp
	}

	// Found everything, or giving up. Add the new imports and remove any unused.
	var fixes []*importFix
	for _, imp := range existingImports {
		// We deliberately ignore globals here, because we can't be sure
		// they're in the same package. People do things like put multiple
		// main packages in the same directory, and we don't want to
		// remove imports if they happen to have the same name as a var in
		// a different package.
		if _, ok := refs[p.importIdentifier(imp)]; !ok {
			fixes = append(fixes, &importFix{StmtInfo: *imp})
			continue
		}
	}

	for _, fix := range fixes {
		if deleted := astutil.DeleteNamedImport(fset, file, fix.StmtInfo.Name, fix.StmtInfo.ImportPath); !deleted {
			return fmt.Errorf("tried to delete import %s and failed", fix.StmtInfo)
		}
	}

	return nil
}

type visitFn func(node ast.Node) ast.Visitor

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}

// An importInfo represents a single import statement.
type importInfo struct {
	ImportPath string // import path, e.g. "crypto/rand".
	Name       string // import name, e.g. "crand", or "" if none.
}

func (i importInfo) String() string {
	return fmt.Sprintf("(local name = %q, path = %q)", i.Name, i.ImportPath)
}

// A pass contains all the inputs and state necessary to fix a file's imports.
// It can be modified in some ways during use; see comments below.
type pass struct {
	knownPackages map[string]*packageInfo // information about all known packages.
}

// A packageInfo represents what's known about a package.
type packageInfo struct {
	name string // real package name, if known.
}

type importFix struct {
	// StmtInfo represents the import statement this fix will add, remove, or change.
	StmtInfo importInfo
}

// collectImports returns all the imports in f.
// Unnamed imports (., _) and "C" are ignored.
func collectImports(f *ast.File) []*importInfo {
	var imports []*importInfo
	for _, imp := range f.Imports {
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if imp.Path.Value == `"C"` || name == "_" || name == "." {
			continue
		}
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, &importInfo{
			Name:       name,
			ImportPath: path,
		})
	}
	return imports
}

// references is set of references found in a Go file. The first map key is the
// left hand side of a selector expression, the second key is the right hand
// side, and the value should always be true.
type references map[string]map[string]bool

// collectReferences builds a map of selector expressions, from
// left hand side (X) to a set of right hand sides (Sel).
func collectReferences(f *ast.File) references {
	refs := references{}

	var visitor visitFn
	visitor = func(node ast.Node) ast.Visitor {
		if node == nil {
			return visitor
		}
		switch v := node.(type) {
		case *ast.SelectorExpr:
			xident, ok := v.X.(*ast.Ident)
			if !ok {
				break
			}
			if xident.Obj != nil {
				// If the parser can resolve it, it's not a package ref.
				break
			}
			if !ast.IsExported(v.Sel.Name) {
				// Whatever this is, it's not exported from a package.
				break
			}
			pkgName := xident.Name
			r := refs[pkgName]
			if r == nil {
				r = make(map[string]bool)
				refs[pkgName] = r
			}
			r[v.Sel.Name] = true
		}
		return visitor
	}
	ast.Walk(visitor, f)
	return refs
}

// importIdentifier returns the identifier that imp will introduce. It will
// guess if the package name has not been loaded, e.g. because the source
// is not available.
func (p *pass) importIdentifier(imp *importInfo) string {
	if imp.Name != "" {
		return imp.Name
	}
	known := p.knownPackages[imp.ImportPath]
	if known != nil && known.name != "" {
		return known.name
	}
	return codegenutil.AssumedPackageName(imp.ImportPath).Name()
}

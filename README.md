# go-codegenutil
Basic code generation utilities for Go metaprogrammers.

The [`codegenutil`
package](https://pkg.go.dev/github.com/meta-programming/go-codegenutil) provides
primitives for representing Go identifiers as `(import path, name)` pairs--the
`*codegenutil.Symbol` type--as well as a `codegenutil.Package` type that
represents a unique name for a Go package.

The [`unusedimport`
package](https://pkg.go.dev/github.com/meta-programming/go-codegenutil/unusedimports)
provides the import pruning features of the `goimports` command in library form
and can be used without depending on the other packages in this library.

The [`codetemplate`
package](https://pkg.go.dev/github.com/meta-programming/go-codegenutil) provides
an alternative to the [`"text/template"`
package](https://pkg.go.dev/text/template) for writing Go code templates. It is
based on a 2022 fork of the `"text/template"` package and uses the same parser.
Such templates can use `*codegenutil.Symbol` values directly as well as a
`"header"` function to insert the package statement and imports blocks.
`codetemplate` will ensure an import exists for each identifier from other
package, and it will format the symbol according to the local name of the
generated import statement.


## Example

```go
import (
    "github.com/meta-programming/go-codegenutil"
    "github.com/meta-programming/go-codegenutil/codetemplate"
)

func example() {
    // Parse the template.
    template, err := codetemplate.Parse("mypkg.go", `// Package mypkg does neat things.
{{header}}

import "automaticallyremoved"

var result1 = {{.maxFn1}}(1, 2)
var result2 = {{.maxFn2}}(1, 2)

func main() {
    fmt.Printf("%f and %f\n", result1, result2)
    fmt.Printf("The people's choice: %f\n", {{.peoplesChoice}})
}
`)
    if err != nil {
        fmt.Printf("error parsing template: %v", err)
        return
    }

    // Execute the template with whatever data you'd like, and use symbols directly.
    filePackage := codegenutil.AssumedPackageName("abc.xyz/mypkg")
    code := &strings.Builder{}
    if err := template.Execute(codegenutil.NewFileImports(filePackage), code, map[string]*codegenutil.Symbol{
        "maxFn1":        codegenutil.Sym("math", "Max"),
        "maxFn2":        codegenutil.Sym("alternative/math", "Max"),
        "peoplesChoice": filePackage.Symbol("result2"),
    }); err != nil {
        fmt.Printf("error executing template: %v", err)
        return
    }

    fmt.Print(code.String())
}
```

Outputs:

```go
// Package mypkg does neat things.
package mypkg

import (
	"math"

	math2 "alternative/math"
)

var result1 = math.Max(1, 2)
var result2 = math2.Max(1, 2)

func main() {
	fmt.Printf("%f and %f\n", result1, result2)
	fmt.Printf("The people's choice: %f\n", result2)
}
```
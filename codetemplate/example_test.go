package codetemplate_test

import (
	"fmt"
	"strings"

	"github.com/meta-programming/go-codegenutil"
	"github.com/meta-programming/go-codegenutil/codetemplate"
)

func Example() {
	template, err := codetemplate.Parse(`// Package mypkg does neat things.
{{header}}

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
	// Output: // Package mypkg does neat things.
	// package mypkg
	//
	// import (
	// 	"math"
	//
	// 	math2 "alternative/math"
	// )
	//
	// var result1 = math.Max(1, 2)
	// var result2 = math2.Max(1, 2)
	//
	// func main() {
	// 	fmt.Printf("%f and %f\n", result1, result2)
	// 	fmt.Printf("The people's choice: %f\n", result2)
	// }
}

func Example_unusedImportPruning() {
	template, err := codetemplate.Parse(`// Package mypkg does neat things.
{{header}}

import (
	// Log isn't used in the output, so the import declaration is deleted.
	"log"
)

var result1 = {{.maxFn1}}(1, 2)
var result2 = {{.maxFn2}}(1, 2)

func main() {
	fmt.Printf("%f and %f\n", result1, result2)
	fmt.Printf("The people's choice: %f\n", {{.peoplesChoice}})
	{{if .neverTrueCondition}}
		log.Printf("this doesn't appear in the output, and the import is pruned")
	{{end}}
}
	`)
	if err != nil {
		fmt.Printf("error parsing template: %v", err)
		return
	}

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
	// Output: // Package mypkg does neat things.
	// package mypkg
	//
	// import (
	// 	"math"
	//
	// 	math2 "alternative/math"
	// )
	//
	// // Log isn't used in the output, so the import declaration is deleted.
	//
	// var result1 = math.Max(1, 2)
	// var result2 = math2.Max(1, 2)
	//
	// func main() {
	// 	fmt.Printf("%f and %f\n", result1, result2)
	// 	fmt.Printf("The people's choice: %f\n", result2)
	//
	// }
}

package astfrom_test

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"

	"github.com/cstockton/astgen/astfrom"
)

func Example() {
	run := func(src string) {
		node := astfrom.Source(src)

		fset := token.NewFileSet()
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			fmt.Println(`Error:`, err)
		}
		fmt.Printf("`%v` ->\n%s\n\n", src, buf.String())
	}

	run(`myIdent`)
	run(`1 + 2`)
	run(`func() {}`)
	run(`var foo = "str"`)
	run(`var foo = "str"; i := 0`)

	// Output:
	// `myIdent` ->
	// myIdent
	//
	// `1 + 2` ->
	// 1 + 2
	//
	// `func() {}` ->
	// func() {
	// }
	//
	// `var foo = "str"` ->
	// var foo = "str"
	//
	// `var foo = "str"; i := 0` ->
	// {
	// 	var foo = "str"
	// 	i := 0
	// }
}

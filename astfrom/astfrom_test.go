package astfrom

import (
	"fmt"
	"go/ast"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const (
	trgExpr  = "_"
	trgDecl  = "_ = " + trgExpr
	trgStmt  = "\t" + trgDecl + "\n"
	trgBlock = "{\n" + trgStmt + "}\n"
	trgFile  = "func " + fnSentinel + " " + trgBlock
	trgPkg   = "package " + pkgSentinel + "\n\n" + trgFile
)

var (
	astExpr   ast.Expr = &ast.Ident{}
	astDecl   ast.Decl = &ast.GenDecl{}
	astStmt   ast.Stmt = &ast.IfStmt{}
	astLit             = &ast.BasicLit{}
	astCall            = &ast.CallExpr{}
	astAssign          = &ast.AssignStmt{}
	astBlock           = &ast.BlockStmt{}
	astFile            = &ast.File{}
)

func TestSource(t *testing.T) {
	type test struct {
		exp       ast.Node
		expReduce ast.Node
		src       string
	}
	tests := []test{
		{astExpr, astExpr, `foo`},
		{astExpr, astExpr, "myIdent"},
		{astExpr, astExpr, `int64`},
		{astLit, astLit, "42"},
		{astCall, astCall, "myIdent()"},
		{astFile, astAssign, "foo := 42"},
		{astFile, astAssign, "{ foo := 42 }"},
		{astFile, astDecl, "type foo string"},
		{astFile, astDecl, `const int = 5`},
		{astFile, astStmt, `if true {};`},
		{astFile, astBlock, `{ var i int64 = 10; s := i+1 };`},
		{astFile, astFile, `package main;`},
		{astFile, astFile,
			`// Package p
			package p
			import "fmt"; func f() { fmt.Println("Hello, World!") };`},
	}
	for idx, test := range tests {
		t.Logf(`test #%va - from src %q exp %[3]T`, idx, test.src, test.exp)

		got, err := source(test.src)
		if err != nil {
			t.Fatalf(`exp nil err from source; got %v`, err)
		}

		expTyp, gotTyp := reflect.TypeOf(test.exp), reflect.TypeOf(got)
		if expTyp != gotTyp {
			t.Fatalf("\n---- [exp] ----\n%v\n\n---- [got] ----\n%v\n", expTyp, gotTyp)
		}

		t.Logf(`test #%vb - from src %q exp %[3]T`, idx, test.src, test.expReduce)
		got = reduce(got)
		expTyp, gotTyp = reflect.TypeOf(test.expReduce), reflect.TypeOf(got)
		if expTyp != gotTyp {
			t.Fatalf("\n---- [exp] ----\n%v\n\n---- [got] ----\n%v\n", expTyp, gotTyp)
		}

	}
}

func TestHeuristics(t *testing.T) {
	type test struct {
		from, to target
		src      string
		exp      string
	}
	tests := []test{
		{targetExpr, targetExpr, "", "_"},
		{targetExpr, targetExpr, "myIdent", "myIdent"},
		{targetExpr, targetExpr, "42", "42"},
		{targetExpr, targetExpr, "myIdent()", "myIdent()"},
		{targetExpr, targetExpr, "myPkg.myIdent", "myPkg.myIdent"},
		{targetExpr, targetExpr, "foo := 42", "foo := 42"},
		{targetExpr, targetDecl, "", "_ = _"},
		{targetDecl, targetDecl, "_ = myIdent", "_ = myIdent"},
		{targetExpr, targetDecl, "_", trgDecl},
		{targetDecl, targetDecl, trgDecl, trgDecl},
		{targetExpr, targetStmt, "", trgStmt},
		{targetDecl, targetStmt, trgDecl, trgStmt},
		{targetStmt, targetStmt, trgStmt, trgStmt},
		{targetExpr, targetBlock, "", trgBlock},
		{targetDecl, targetBlock, trgDecl, trgBlock},
		{targetStmt, targetBlock, trgStmt, trgBlock},
		{targetBlock, targetBlock, trgBlock, trgBlock},
		{targetExpr, targetFile, "", trgFile},
		{targetDecl, targetFile, trgDecl, trgFile},
		{targetStmt, targetFile, trgStmt, trgFile},
		{targetBlock, targetFile, trgBlock, trgFile},
		{targetFile, targetFile, trgFile, trgFile},
		{targetExpr, targetPkg, "", trgPkg},
		{targetDecl, targetPkg, trgDecl, trgPkg},
		{targetStmt, targetPkg, trgStmt, trgPkg},
		{targetBlock, targetPkg, trgBlock, trgPkg},
		{targetFile, targetPkg, trgFile, trgPkg},
		{targetPkg, targetPkg, trgPkg, trgPkg},
	}
	for idx, test := range tests {
		t.Logf(`test #%v - for from %v to %v with src %q`,
			idx, test.from, test.to, test.src)
		exp, grown := test.exp, expand(test.src, test.from, test.to)
		if exp != grown {
			t.Fatalf("\n---- [exp] ----\n%v\n\n---- [got] ----\n%v\n", exp, grown)
		}

		node, err := source(grown)
		if err != nil {
			t.Fatalf(`exp nil err from Parse, got %v`, err)
		}

		reduced := reduce(node)
		if strings.Contains(grown, fnSentinelName) {
			// sentinel func, check we reduced ast to the base ident
			id, ok := reduced.(*ast.Ident)
			if !ok {
				t.Fatalf(`exp reduce to ast.Ident; got %v (%[1]T)`, reduced)
			}
			if id.Name != "_" {
				t.Fatalf(`exp reduce to blank ident; got %v (%[1]T)`, reduced)
			}
		}
	}
}

func TestTarget(t *testing.T) {
	t.Run(`String`, func(t *testing.T) {
		type test struct {
			trg target
			exp string
		}
		tests := []test{
			{targetNode, "Node"},
			{targetBlock, "Block"},
			{targetDecl, "Decl"},
			{targetExpr, "Expr"},
			{targetFile, "File"},
			{targetPkg, "Pkg"},
			{targetStmt, "Stmt"},

			// oob/ob1
			{targetNode - 1, "Node"}, {targetNode - 2, "Node"},
			{targetPkg + 1, "Node"}, {targetPkg + 2, "Node"},
		}
		for idx, test := range tests {
			t.Logf(`test #%v - exp %v from node %d (%[3]v)`, idx, test.exp, test.trg)
			if exp, got := test.exp, test.trg.String(); exp != got {
				t.Fatalf(`exp Target String() to return %q; got %q`, exp, got)
			}
		}
	})
}

func TestRecoverFn(t *testing.T) {
	t.Run("CallsFunc", func(t *testing.T) {
		var called bool

		err := recoverFn(func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Error("expected no error in recoverFn()")
		}
		if !called {
			t.Error("Expected recoverFn() to call func")
		}
	})
	t.Run("PropagatesError", func(t *testing.T) {
		err := fmt.Errorf("expect this error")
		rerr := recoverFn(func() error {
			return err
		})
		if err != rerr {
			t.Error("expected recoverFn() to propagate")
		}
	})
	t.Run("PropagatesPanicError", func(t *testing.T) {
		err := fmt.Errorf("expect this error")
		rerr := recoverFn(func() error {
			panic(err)
		})
		if err != rerr {
			t.Error("Expected recoverFn() to propagate")
		}
	})
	t.Run("PropagatesRuntimeError", func(t *testing.T) {
		err := recoverFn(func() error {
			sl := []int{}
			_ = sl[0]
			return nil
		})
		if err == nil {
			t.Error("expected runtime error to propagate")
		}
		if _, ok := err.(runtime.Error); !ok {
			t.Error("expected runtime error to retain type type")
		}
	})
	t.Run("PropagatesString", func(t *testing.T) {
		exp := "panic: string type panic"
		rerr := recoverFn(func() error {
			panic("string type panic")
		})
		if exp != rerr.Error() {
			t.Errorf("expected recoverFn() to return %v, got: %v", exp, rerr)
		}
	})
}

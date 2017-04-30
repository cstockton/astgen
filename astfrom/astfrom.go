// Package astfrom provides a more forgiving interface for ad-hoc creation of
// objects in the go/ast package. It's intended for quick debugging and
// inspection of Go ASTs. See the parent package for a higher level interface
// for generating complete ASTs at runtime.
package astfrom

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Source will return a valid ast.Node from all well formed Go source code. The
// returned node will never be nil, instead returning a simple *ast.Ident
// containing the error string if a failure occurs.
func Source(src string) ast.Node {
	node, err := source(src)
	if err != nil {
		return errIdent(err)
	}
	return reduce(node)
}

func source(src string) (ast.Node, error) {
	var (
		err  error
		node ast.Node
	)
	for cur, from := src, targetExpr; from <= targetPkg; from++ {
		switch from {
		case targetExpr:
			err = recoverFn(func() (err error) {
				node, err = parser.ParseExpr(cur)
				return err
			})
		default:
			err = recoverFn(func() (err error) {
				node, err = parser.ParseFile(token.NewFileSet(), `string.go`, cur, 0)
				return err
			})
		}
		if err == nil {
			break
		}
		cur = expand(src, from+1, targetPkg)
	}
	if err != nil {
		return nil, err
	}
	return node, nil
}

const (
	pkgSentinel    = `astfrom`
	fnSentinelName = `astfromFunc`
	fnSentinel     = fnSentinelName + `()`
)

func expand(src string, from, to target) string {
	src = expandExpr(src, from, to)
	src = expandFile(src, from, to)
	return src
}

func expandExpr(src string, from, to target) string {
	if len(src) == 0 {
		src = `_`
	}
	if to >= targetDecl && targetDecl > from {
		src = "_ = " + src
	}
	if to >= targetStmt && targetStmt > from {
		src = "\t" + src + "\n"
	}
	return src
}

func expandFile(src string, from, to target) string {
	if to >= targetBlock && targetBlock > from {
		src = "{\n" + strings.TrimRight(src, "\n\t") + "\n}\n"
	}
	if to >= targetFile && targetFile > from && from <= targetBlock {
		src = "func " + fnSentinel + " " + src
	}
	if to >= targetPkg && targetPkg > from {
		src = "package " + pkgSentinel + "\n\n" + src
	}
	return src
}

func reduce(node ast.Node) ast.Node {
	switch T := node.(type) {
	case *ast.File:
		if T.Name.Name == pkgSentinel {
			return reduce(T.Decls[0])
		}
	case *ast.FuncDecl:
		if T.Name.Name == fnSentinelName {
			return reduce(T.Body)
		}
	case *ast.BlockStmt:
		if len(T.List) == 1 {
			return reduce(T.List[0])
		}
	case *ast.DeclStmt:
		return T.Decl
	case *ast.AssignStmt:
		id, ok := T.Lhs[0].(*ast.Ident)
		if ok && len(T.Lhs) == 1 && id.Name == "_" {
			return T.Rhs[0]
		}
	}
	return node
}

// target specifies the target node type.
type target int

// The available target modes, ordered in smallest to largest.
const (
	targetNode target = iota
	targetExpr
	targetDecl
	targetStmt
	targetBlock
	targetFile
	targetPkg
)

var targetStrings = [...]string{
	targetNode:  "Node",
	targetExpr:  "Expr",
	targetDecl:  "Decl",
	targetStmt:  "Stmt",
	targetBlock: "Block",
	targetFile:  "File",
	targetPkg:   "Pkg",
}

// String implements fmt.Stringer by returning the name of the target.
func (s target) String() string {
	if targetNode > s || s > targetPkg {
		s = targetNode
	}
	return targetStrings[s]
}

// errIdent will return an *ast.Ident to represent the given error in place of
// nil, a *Bad(Expr|Stmt|Decl) node or producing a panic.
func errIdent(err error) *ast.Ident {
	return ast.NewIdent(err.Error())
}

// recoverFn will attempt to execute f, if f return a non-nil error it will be
// returned. If f panics this function will attempt to recover() and return a
// error instead.
func recoverFn(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch T := r.(type) {
			case error:
				err = T
			default:
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()
	err = f()
	return
}

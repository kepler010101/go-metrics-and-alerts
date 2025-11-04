package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	panicMsg = "panic call is not allowed"
	fatalMsg = "log.Fatal is not allowed outside main"
	exitMsg  = "os.Exit is not allowed outside main"
)

var Analyzer = &analysis.Analyzer{
	Name: "projectlinter",
	Doc:  "reports panic usage and restricted fatal or exit calls",
	Run:  run,
}

func main() {
	singlechecker.Main(Analyzer)
}

type funcContext struct {
	inMain bool
}

func run(pass *analysis.Pass) (interface{}, error) {
	isMainPkg := pass.Pkg.Name() == "main"

	for _, file := range pass.Files {
		var stack []funcContext

		astutil.Apply(file, func(c *astutil.Cursor) bool {
			switch node := c.Node().(type) {
			case *ast.FuncDecl:
				ctx := funcContext{}
				if isMainPkg && node.Recv == nil && node.Name != nil && node.Name.Name == "main" {
					ctx.inMain = true
				}
				stack = append(stack, ctx)
			case *ast.FuncLit:
				stack = append(stack, funcContext{})
			case *ast.CallExpr:
				checkCall(pass, node, stack)
			}
			return true
		}, func(c *astutil.Cursor) bool {
			switch c.Node().(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
			return true
		})
	}

	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, stack []funcContext) {
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
		if obj, ok := pass.TypesInfo.Uses[ident]; !ok || obj == types.Universe.Lookup("panic") {
			pass.Reportf(ident.Pos(), panicMsg)
		}
		return
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	obj, ok := pass.TypesInfo.Uses[sel.Sel]
	if !ok || obj == nil {
		return
	}

	pkg := obj.Pkg()

	ctx := currentContext(stack)

	if pkg != nil && pkg.Path() == "log" && strings.HasPrefix(obj.Name(), "Fatal") {
		if !ctx.inMain {
			pass.Reportf(sel.Sel.Pos(), fatalMsg)
		}
		return
	}

	if pkg != nil && pkg.Path() == "os" && obj.Name() == "Exit" {
		if !ctx.inMain {
			pass.Reportf(sel.Sel.Pos(), exitMsg)
		}
	}
}

func currentContext(stack []funcContext) funcContext {
	if len(stack) == 0 {
		return funcContext{}
	}
	return stack[len(stack)-1]
}

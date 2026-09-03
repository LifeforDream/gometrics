// Package exitanalyzer реализует проверку на:
// - использование panic() в коде приложения
// - использование log.Fatal() и os.Exit() вне функции main пакета main
package exitanalyzer

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// ExitAnalyzer представляет собой экспортируемый анализатор
var ExitAnalyzer = &analysis.Analyzer{
	Name: "exitanalyzer",
	Doc:  "check for panic() calls and log.Fatal() and os.Exit() outside main",
	Run:  run,
}

const (
	// panicLit - зарезервированный литерал panic
	panicLit = "panic"
	// mainLit - зарезервированный литерал main
	mainLit = "main"
)

type forbiddenCall struct {
	pkg, fn string
}

var forbiddenCalls = []forbiddenCall{
	{"log", "Fatal"},
	{"os", "Exit"},
}

func isPanicStmt(call *ast.CallExpr) bool {
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if ident.Name == panicLit {
			return true
		}
	}
	return false
}

func findPanics(pass *analysis.Pass, file *ast.File) {
	if strings.HasSuffix(pass.Fset.File(file.Pos()).Name(), "_test.go") {
		return
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.ExprStmt:
			if call, ok := x.X.(*ast.CallExpr); ok {
				if isPanicStmt(call) {
					pass.Reportf(x.Pos(), "panic usage")
				}
			}
		case *ast.GoStmt:
			if isPanicStmt(x.Call) {
				pass.Reportf(x.Pos(), "panic usage in go statement")
			}
		case *ast.DeferStmt:
			if isPanicStmt(x.Call) {
				pass.Reportf(x.Pos(), "panic usage in defer statement")
			}
		}
		return true
	})
}

// mainBody возвращает тело функции main пакета main, если file принадлежит
// пакету main, иначе nil.
func mainBody(file *ast.File) *ast.BlockStmt {
	if file.Name.Name != mainLit {
		return nil
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.Name == mainLit {
			return fd.Body
		}
	}
	return nil
}

func findLogExits(pass *analysis.Pass, file *ast.File) {
	exempt := mainBody(file)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		obj := pass.TypesInfo.Uses[ident]
		pkgName, ok := obj.(*types.PkgName)
		if !ok {
			// ident не является именем пакета
			return true
		}
		importPath := pkgName.Imported().Path() // реальная ссылка на импортированный пакет
		// вызов находится внутри тела main.main - исключение
		if exempt != nil && exempt.Pos() <= call.Pos() && call.Pos() < exempt.End() {
			return true
		}
		for _, fc := range forbiddenCalls {
			if importPath == fc.pkg && sel.Sel.Name == fc.fn {
				pass.Reportf(sel.Pos(), "found %s.%s usage outside main.main", ident.Name, sel.Sel.Name)
			}
		}
		return true
	})
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		findPanics(pass, file)
		findLogExits(pass, file)
	}
	return nil, nil
}

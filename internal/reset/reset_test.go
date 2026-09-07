package reset

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseTypeDecl разбирает src как тело файла (заголовок package
// добавляется автоматически) и возвращает первый *ast.GenDecl с Tok ==
// token.TYPE — для изолированной проверки hasResetDirective.
func parseTypeDecl(t *testing.T, src string) *ast.GenDecl {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", "package p\n\n"+src, parser.ParseComments)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			return gd
		}
	}
	t.Fatalf("no type declaration found in source:\n%s", src)
	return nil
}

func TestHasResetDirective(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		specIndex  int
		wantResult bool
	}{
		{
			name: "standalone decl with directive",
			src: `
// generate:reset
type Foo struct {
	X int
}`,
			specIndex:  0,
			wantResult: true,
		},
		{
			name: "standalone decl without directive",
			src: `
type Foo struct {
	X int
}`,
			specIndex:  0,
			wantResult: false,
		},
		{
			name: "standalone decl with unrelated comment",
			src: `
// обычная структура
type Foo struct {
	X int
}`,
			specIndex:  0,
			wantResult: false,
		},
		{
			name: "grouped decl, directive on first spec only",
			src: `
type (
	// generate:reset
	Bar struct {
		Y string
	}

	Baz struct {
		Z string
	}
)`,
			specIndex:  0,
			wantResult: true,
		},
		{
			name: "grouped decl, spec without its own directive is not affected by sibling's",
			src: `
type (
	// generate:reset
	Bar struct {
		Y string
	}

	Baz struct {
		Z string
	}
)`,
			specIndex:  1,
			wantResult: false,
		},
		{
			name: "grouped decl with group-level doc must not leak onto a doc-less spec",
			src: `
// комментарий к группе, не директива ни для одного из членов
type (
	Qux struct{ A int }

	// generate:reset
	Quux struct{ B int }
)`,
			specIndex:  0,
			wantResult: false,
		},
		{
			name: "grouped decl with group-level doc, targeted spec still matches via its own doc",
			src: `
// комментарий к группе, не директива ни для одного из членов
type (
	Qux struct{ A int }

	// generate:reset
	Quux struct{ B int }
)`,
			specIndex:  1,
			wantResult: true,
		},
		{
			name: "single-spec group: directive attaches to the spec, not the group",
			src: `
type (
	// generate:reset
	Solo struct{ C int }
)`,
			specIndex:  0,
			wantResult: true,
		},
		{
			name: "directive text must match exactly, trailing text disqualifies it",
			src: `
// generate:reset пожалуйста
type Foo struct {
	X int
}`,
			specIndex:  0,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gd := parseTypeDecl(t, tt.src)
			ts := gd.Specs[tt.specIndex].(*ast.TypeSpec)

			got := hasResetDirective(gd, ts, len(gd.Specs))

			assert.Equal(t, tt.wantResult, got)
		})
	}
}

func TestResetTargets(t *testing.T) {
	const src = `
package p

// generate:reset
type Foo struct {
	X int
}

type Bar struct {
	Y int
}

// generate:reset
type Baz int

type (
	Qux struct{ A int }

	// generate:reset
	Quux struct{ B int }
)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	require.NoError(t, err)

	targets := resetTargets(file)

	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.ts.Name.Name)
	}

	assert.Equal(t, []string{"Foo", "Quux"}, names)
}

// copyModuleDir копирует все .go-файлы из testdata/<name> во временный
// каталог и превращает его в самостоятельный Go-модуль, чтобы run() из
// cmd/reset можно было запустить на нём, не трогая исходную фикстуру и не
// завися от go.mod самого репозитория.
func copyModuleDir(t *testing.T, name string) string {
	t.Helper()

	srcDir := filepath.Join("testdata", name)
	entries, err := os.ReadDir(srcDir)
	require.NoError(t, err)

	dir := t.TempDir()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, entry.Name()), src, 0o644))
	}

	goMod := "module fixture\n\ngo 1.23\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644))

	return dir
}

// copyFixtureModule копирует testdata/fixture — см. copyModuleDir.
func copyFixtureModule(t *testing.T) string {
	t.Helper()
	return copyModuleDir(t, "fixture")
}

// Отдельной проверки "Run генерирует go/parser-валидный файл" нет: это
// строго слабее того, что доказывает сам факт компиляции и прохождения
// поведенческого теста ниже.
func TestRun_GeneratedResetMethodBehavesCorrectly(t *testing.T) {
	dir := copyFixtureModule(t)

	require.NoError(t, Run(dir))

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated reset.gen.go failed its behavioral test:\n%s", out)
}

func TestRun_SkipsExistingResetGenGo(t *testing.T) {
	dir := copyFixtureModule(t)
	require.NoError(t, Run(dir))

	// Повторный запуск не должен спотыкаться о результат предыдущего
	// запуска (например, находя ложное совпадение "generate:reset" внутри
	// reset.gen.go или дублируя объявления методов).
	err := Run(dir)
	require.NoError(t, err)

	genPath := filepath.Join(dir, "reset.gen.go")
	src, err := os.ReadFile(genPath)
	require.NoError(t, err)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, genPath, src, 0)
	require.NoError(t, err)

	count := 0
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Name.Name == "Reset" {
			count++
		}
	}
	assert.Equal(t, 1, count, "Reset() must be generated exactly once for ResetableStruct")
}

// TestRun_RejectsConflictingImportAliases проверяет, что два поля из разных
// файлов одного пакета, ссылающиеся на один и тот же путь импорта под
// разными алиасами, приводят к ошибке, а не к тихому выбору одного из них.
func TestRun_RejectsConflictingImportAliases(t *testing.T) {
	dir := copyModuleDir(t, "importconflict")

	err := Run(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple aliased import statements")
}

// TestLoadPackages_ReturnsPackageErrors проверяет, что ошибки, попавшие в
// pkg.Errors при загрузке (здесь — синтаксическая ошибка в testdata/broken),
// пробрасываются из loadPackages как error, а не игнорируются. Это
// единственный путь loadPackages, не задействованный ни одним из TestRun_*
// (все их фикстуры валидны) — успешная загрузка уже покрыта ими напрямую,
// отдельный тест на неё был бы чистым дублированием без нового сигнала.
func TestLoadPackages_ReturnsPackageErrors(t *testing.T) {
	dir := copyModuleDir(t, "broken")

	_, err := loadPackages(dir)

	require.Error(t, err)
}

// typeCheckFile разбирает src (полноценный файл пакета p, с заголовком и
// нужными импортами) и type-checks его через go/types с обычным Importer.
// Даёт *ast.File, *types.Info и *types.Package для прямого юнит-тестирования
// genReset/hasResetMethod/generateStar/requiredImport — без обращения к
// packages.Load, диску и временным модулям.
func typeCheckFile(t *testing.T, src string) (*ast.File, *types.Info, *types.Package) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	require.NoError(t, err)

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	require.NoError(t, err)

	return file, info, pkg
}

// fieldType возвращает ast.Expr типа поля fieldName структуры structName,
// объявленной в file.
func fieldType(t *testing.T, file *ast.File, structName, fieldName string) ast.Expr {
	t.Helper()

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			if ts.Name.Name != structName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			require.True(t, ok, "%s is not a struct", structName)
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					if name.Name == fieldName {
						return field.Type
					}
				}
			}
		}
	}
	t.Fatalf("field %s.%s not found", structName, fieldName)
	return nil
}

// TestGenReset покрывает все ветки switch по typ.(type): примитив, именованный
// слайс/мапу и сырые []T/[N]T/map[K]V, а также делегирование в Reset() для
// значимого локального типа и типа из другого пакета (bytes.Buffer).
func TestGenReset(t *testing.T) {
	const src = `package p

import (
	"bytes"
	"time"
)

type MyNamed struct{ X int }

func (m *MyNamed) Reset() {}

type namedSlice []int

type namedMap map[string]int

type S struct {
	I     int
	Sl    []int
	Arr   [3]int
	M     map[string]int
	NSl   namedSlice
	NM    namedMap
	Named MyNamed
	Buf   bytes.Buffer
	T     time.Time
}
`
	file, info, _ := typeCheckFile(t, src)

	tests := []struct {
		field string
		want  string
	}{
		{"I", "x = *new(int)\n"},
		{"Sl", "x = x[:0]\n"},
		{"Arr", "x = *new([3]int)\n"},
		{"M", "clear(x)\n"},
		{"NSl", "x = x[:0]\n"},
		{"NM", "clear(x)\n"},
		{"Named", "x.Reset()\n"},
		{"Buf", "x.Reset()\n"},
		{"T", "x = *new(time.Time)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			typ := fieldType(t, file, "S", tt.field)
			var sb strings.Builder
			err := genReset(&sb, "x", typ, make(map[string]string), targetSpec{file: file}, info, false)
			require.NoError(t, err)
			assert.Equal(t, tt.want, sb.String())
		})
	}

	t.Run("value-typed field of a type from another package registers its import", func(t *testing.T) {
		typ := fieldType(t, file, "S", "T")
		var sb strings.Builder
		imports := make(map[string]string)
		require.NoError(t, genReset(&sb, "x", typ, imports, targetSpec{file: file}, info, false))
		assert.Equal(t, map[string]string{"time": ""}, imports)
	})

	t.Run("delegating to Reset() does not need the package import", func(t *testing.T) {
		typ := fieldType(t, file, "S", "Buf")
		var sb strings.Builder
		imports := make(map[string]string)
		require.NoError(t, genReset(&sb, "x", typ, imports, targetSpec{file: file}, info, false))
		assert.Empty(t, imports, "rs.field.Reset() never spells out the package name")
	})
}

// TestGenReset_StarExpr проверяет ветку *ast.StarExpr: и указатель на
// примитив, и указатель на тип со своим Reset() должны давать рантайм-
// проверку "resetter, ok := ...".
func TestGenReset_StarExpr(t *testing.T) {
	file, info, _ := typeCheckFile(t, `package p

type MyNamed struct{ X int }

func (m *MyNamed) Reset() {}

type S struct {
	P  *int
	PN *MyNamed
}
`)

	t.Run("pointer to primitive", func(t *testing.T) {
		typ := fieldType(t, file, "S", "P")
		var sb strings.Builder
		err := genReset(&sb, "x", typ, make(map[string]string), targetSpec{file: file}, info, false)
		require.NoError(t, err)
		out := sb.String()
		assert.Contains(t, out, "if x != nil {")
		assert.Contains(t, out, "any(x).(interface{ Reset() })")
		assert.Contains(t, out, "*x = *new(int)")
	})

	t.Run("pointer to Reset()-able struct", func(t *testing.T) {
		typ := fieldType(t, file, "S", "PN")
		var sb strings.Builder
		err := genReset(&sb, "x", typ, make(map[string]string), targetSpec{file: file}, info, false)
		require.NoError(t, err)
		out := sb.String()
		assert.Contains(t, out, "if x != nil {")
		assert.Contains(t, out, "resetter.Reset()")
		// "мёртвая" ветка else всё равно должна быть валидным Go, а не
		// "*x.Reset()" (см. doc genReset про skipDelegation).
		assert.Contains(t, out, "*x = *new(MyNamed)")
		assert.NotContains(t, out, "*x.Reset()")
	})
}

// TestGenReset_AllowsSharedImportAcrossFiles проверяет, что два поля из
// разных файлов одного пакета, ссылающиеся на один и тот же путь импорта под
// одинаковым (в т.ч. пустым) алиасом, не считаются конфликтом.
func TestGenReset_AllowsSharedImportAcrossFiles(t *testing.T) {
	fileA, infoA, _ := typeCheckFile(t, `package p

import "time"

type A struct{ T time.Time }
`)
	fileB, infoB, _ := typeCheckFile(t, `package p

import "time"

type B struct{ T time.Time }
`)

	imports := make(map[string]string)
	var sbA strings.Builder
	require.NoError(t, genReset(&sbA, "a", fieldType(t, fileA, "A", "T"), imports, targetSpec{file: fileA}, infoA, false))

	var sbB strings.Builder
	err := genReset(&sbB, "b", fieldType(t, fileB, "B", "T"), imports, targetSpec{file: fileB}, infoB, false)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"time": ""}, imports)
}

// TestGenReset_PropagatesImportConflictError проверяет, что genReset
// возвращает ошибку конфликта алиасов, если один и тот же путь импорта уже
// зарегистрирован в imports под другим алиасом.
func TestGenReset_PropagatesImportConflictError(t *testing.T) {
	fileA, infoA, _ := typeCheckFile(t, `package p

import atime "time"

type A struct{ T atime.Time }
`)
	fileB, infoB, _ := typeCheckFile(t, `package p

import "time"

type B struct{ T time.Time }
`)

	imports := make(map[string]string)
	var sbA strings.Builder
	require.NoError(t, genReset(&sbA, "a", fieldType(t, fileA, "A", "T"), imports, targetSpec{file: fileA}, infoA, false))

	var sbB strings.Builder
	err := genReset(&sbB, "b", fieldType(t, fileB, "B", "T"), imports, targetSpec{file: fileB}, infoB, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple aliased import statements")
}

// TestGenReset_PropagatesImportConflictErrorThroughStar проверяет тот же
// конфликт, но когда он всплывает из рекурсивного вызова genReset внутри
// generateStar (поле-указатель на тип из другого пакета) — ошибка должна
// дойти до вызывающего через обёртку "error while generating star expression".
func TestGenReset_PropagatesImportConflictErrorThroughStar(t *testing.T) {
	fileA, infoA, _ := typeCheckFile(t, `package p

import atime "time"

type A struct{ T *atime.Time }
`)
	fileB, infoB, _ := typeCheckFile(t, `package p

import "time"

type B struct{ T *time.Time }
`)

	imports := make(map[string]string)
	var sbA strings.Builder
	require.NoError(t, genReset(&sbA, "a", fieldType(t, fileA, "A", "T"), imports, targetSpec{file: fileA}, infoA, false))

	var sbB strings.Builder
	err := genReset(&sbB, "b", fieldType(t, fileB, "B", "T"), imports, targetSpec{file: fileB}, infoB, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error while generating star expression")
	assert.Contains(t, err.Error(), "multiple aliased import statements")
}

// TestHasResetMethod проверяет, что учитываются только методы Reset() без
// параметров и результатов, независимо от того, указательный у них ресивер
// или значимый.
func TestHasResetMethod(t *testing.T) {
	const src = `package p

type NoReset struct{}

type PtrReset struct{}

func (r *PtrReset) Reset() {}

type ValReset struct{}

func (r ValReset) Reset() {}

type ParamReset struct{}

func (r *ParamReset) Reset(x int) {}

type ResultReset struct{}

func (r *ResultReset) Reset() int { return 0 }
`
	_, _, pkg := typeCheckFile(t, src)

	namedType := func(name string) types.Type {
		obj := pkg.Scope().Lookup(name)
		require.NotNil(t, obj, "type %s not found", name)
		return obj.Type()
	}

	tests := []struct {
		typeName string
		want     bool
	}{
		{"NoReset", false},
		{"PtrReset", true},
		{"ValReset", true},
		{"ParamReset", false},
		{"ResultReset", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			assert.Equal(t, tt.want, hasResetMethod(namedType(tt.typeName)))
		})
	}
}

// TestGeneratePrimitive, TestGenerateSlice, TestGenerateMap проверяют
// низкоуровневые форматтеры кода сброса напрямую.
func TestGeneratePrimitive(t *testing.T) {
	assert.Equal(t, "rs.X = *new(int)", generatePrimitive("rs.X", ast.NewIdent("int")))
}

func TestGenerateSlice(t *testing.T) {
	assert.Equal(t, "rs.S = rs.S[:0]", generateSlice("rs.S"))
}

func TestGenerateMap(t *testing.T) {
	assert.Equal(t, "clear(rs.M)", generateMap("rs.M"))
}

// TestGenerateStar проверяет generateStar напрямую (не через genReset):
// рантайм-проверка Reset() генерируется всегда, а "мёртвая" ветка else —
// это обычное обнуление через genReset со skipDelegation=true.
func TestGenerateStar(t *testing.T) {
	file, info, _ := typeCheckFile(t, `package p

type MyNamed struct{ X int }

func (m *MyNamed) Reset() {}

type S struct {
	P  *int
	PN *MyNamed
}
`)

	t.Run("pointee without Reset() falls back to zeroing", func(t *testing.T) {
		star := fieldType(t, file, "S", "P").(*ast.StarExpr)
		code, err := generateStar("x", star.X, make(map[string]string), targetSpec{file: file}, info)
		require.NoError(t, err)
		assert.Equal(t, "if x != nil {\nif resetter, ok := any(x).(interface{ Reset() }); ok {\nresetter.Reset()\n} else {\n*x = *new(int)\n}\n}\n", code)
	})

	t.Run("pointee with Reset() still compiles in the dead else branch", func(t *testing.T) {
		star := fieldType(t, file, "S", "PN").(*ast.StarExpr)
		code, err := generateStar("x", star.X, make(map[string]string), targetSpec{file: file}, info)
		require.NoError(t, err)
		assert.Contains(t, code, "resetter.Reset()")
		assert.Contains(t, code, "*x = *new(MyNamed)")
	})

	t.Run("propagates the recursive genReset error", func(t *testing.T) {
		fileA, infoA, _ := typeCheckFile(t, `package p

import atime "time"

type A struct{ T *atime.Time }
`)
		fileB, infoB, _ := typeCheckFile(t, `package p

import "time"

type B struct{ T *time.Time }
`)

		imports := make(map[string]string)
		starA := fieldType(t, fileA, "A", "T").(*ast.StarExpr)
		_, err := generateStar("a", starA.X, imports, targetSpec{file: fileA}, infoA)
		require.NoError(t, err)

		starB := fieldType(t, fileB, "B", "T").(*ast.StarExpr)
		_, err = generateStar("b", starB.X, imports, targetSpec{file: fileB}, infoB)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple aliased import statements")
	})
}

// TestRequiredImport проверяет три исхода: импорт найден по алиасу, найден
// по базовому имени пути (без алиаса), и не найден вовсе.
func TestRequiredImport(t *testing.T) {
	const src = `package p

import (
	atime "time"
	"bytes"
)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	require.NoError(t, err)

	t.Run("found with explicit alias", func(t *testing.T) {
		ipath, alias, ok := requiredImport(file, "atime")
		assert.True(t, ok)
		assert.Equal(t, "time", ipath)
		assert.Equal(t, "atime", alias)
	})

	t.Run("found without alias, matched by base name", func(t *testing.T) {
		ipath, alias, ok := requiredImport(file, "bytes")
		assert.True(t, ok)
		assert.Equal(t, "bytes", ipath)
		assert.Equal(t, "", alias)
	})

	t.Run("not found", func(t *testing.T) {
		_, _, ok := requiredImport(file, "nonexistent")
		assert.False(t, ok)
	})

	t.Run("aliased import does not match its own base name", func(t *testing.T) {
		// "time" здесь импортирован только под алиасом atime, поэтому
		// поиск по базовому имени "time" не должен его найти.
		_, _, ok := requiredImport(file, "time")
		assert.False(t, ok)
	})
}

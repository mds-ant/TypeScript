package checker_test

import (
	"path/filepath"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/repo"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
	"gotest.tools/v3/assert"
)

func TestGetSymbolAtLocation(t *testing.T) {
	t.Parallel()

	content := `interface Foo {
  bar: string;
}
declare const foo: Foo;
foo.bar;`
	fs := vfstest.FromMap(map[string]string{
		"/foo.ts": content,
		"/tsconfig.json": `
				{
					"compilerOptions": {},
					"files": ["foo.ts"]
				}
			`,
	}, tspath.CaseInsensitive /*caseSensitivity*/)
	fs = bundled.WrapFS(fs)

	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, fs, nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()
	file := p.GetSourceFile("/foo.ts")
	interfaceId := file.Statements.Nodes[0].Name()
	varId := file.Statements.Nodes[1].AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes[0].Name()
	propAccess := file.Statements.Nodes[2].Expression()
	nodes := []*ast.Node{interfaceId, varId, propAccess}
	for _, node := range nodes {
		symbol := c.GetSymbolAtLocation(node)
		if symbol == nil {
			t.Fatalf("Expected symbol to be non-nil")
		}
	}
}

func TestGetTypeAtLocationOfTypeOnlyImportClause(t *testing.T) {
	t.Parallel()

	fs := vfstest.FromMap(map[string]string{
		"/types.ts": `export type U = number;
export default interface D { x: number }`,
		"/main.ts": `import type { U } from "./types";
import type * as types from "./types";
import { U as V } from "./types";
import type D from "./types";
export const u: U = 1;
export const v: V = 1;
export type W = types.U;
export type E = D;`,
		"/tsconfig.json": `
				{
					"compilerOptions": {},
					"files": ["types.ts", "main.ts"]
				}
			`,
	}, tspath.CaseInsensitive)
	fs = bundled.WrapFS(fs)

	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)

	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, fs, nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()
	file := p.GetSourceFile("/main.ts")
	importClauseAt := func(index int) *ast.Node {
		return file.Statements.Nodes[index].AsImportDeclaration().ImportClause
	}
	// An import clause without a default binding has no symbol of its own. A type-only one
	// should get the same type as the equivalent regular import instead of crashing.
	regular := c.GetTypeAtLocation(importClauseAt(2))
	for _, index := range []int{0, 1} {
		typ := c.GetTypeAtLocation(importClauseAt(index))
		if typ == nil {
			t.Fatalf("Expected type of import clause %d to be non-nil", index)
		}
		assert.Equal(t, typ, regular)
	}

	defaultClause := c.GetTypeAtLocation(importClauseAt(3))
	defaultReference := c.GetTypeAtLocation(file.Statements.Nodes[7].AsTypeAliasDeclaration().Type)
	assert.Equal(t, defaultClause, defaultReference)
}

func TestCompareSymbolsOrdersModuleSymbolBeforeNamespaceImportClone(t *testing.T) {
	t.Parallel()

	fs := vfstest.FromMap(map[string]string{
		"/defaults.ts": `export const retries = 3;
export default { retries };`,
		"/use.ts": `import * as defaults from "./defaults";
declare const override: typeof import("./defaults");
declare const useOverride: boolean;
export const config = useOverride ? override : defaults;`,
		"/tsconfig.json": `
				{
					"compilerOptions": { "module": "esnext", "moduleResolution": "bundler" },
					"files": ["defaults.ts", "use.ts"]
				}
			`,
	}, tspath.CaseInsensitive)
	fs = bundled.WrapFS(fs)

	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, fs, nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config:         parsed,
		Host:           host,
		SingleThreaded: core.TSTrue,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()

	// Mint the shared symbol's ID from a block reserved after the checker's own, as another checker can.
	moduleSymbol := p.GetSourceFile("/defaults.ts").Symbol
	var other ast.IdAllocator
	other.GetSymbolId(moduleSymbol)

	statements := p.GetSourceFile("/use.ts").Statements.Nodes
	namespaceImport := statements[0].AsImportDeclaration().ImportClause.AsImportClause().NamedBindings
	clone := c.GetTypeAtLocation(namespaceImport.Name()).Symbol()
	assert.Assert(t, clone != moduleSymbol, "Expected the namespace import to have its own symbol")
	assert.Assert(t, ast.GetSymbolId(clone) < ast.GetSymbolId(moduleSymbol), "Expected the clone to have the smaller ID")

	assert.Assert(t, c.CompareSymbols(moduleSymbol, clone) < 0)
	assert.Assert(t, c.CompareSymbols(clone, moduleSymbol) > 0)
	config := statements[3].AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes[0].Name()
	assert.Assert(t, c.GetTypeAtLocation(config).Symbol() == moduleSymbol, "Expected subtype reduction to keep the module symbol's type")
}

func TestSymbolIdsFollowFirstTouchOrderWithinChecker(t *testing.T) {
	t.Parallel()

	fs := vfstest.FromMap(map[string]string{
		"/graphics.d.ts": `declare namespace graphics {
    interface Point { x: number; y: number }
}
declare var graphics: { Point: new (x: number, y: number) => graphics.Point };
declare module "graphics" {
    export = graphics;
}`,
		"/main.ts": `import { Point as P1 } from "graphics";
import { Point as P2 } from "graphics";
import { Point as P3 } from "graphics";
export declare const points: [P1, P2, P3];`,
		"/tsconfig.json": `
				{
					"compilerOptions": { "module": "commonjs", "esModuleInterop": true },
					"files": ["graphics.d.ts", "main.ts"]
				}
			`,
	}, tspath.CaseInsensitive)
	fs = bundled.WrapFS(fs)

	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, fs, nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config:         parsed,
		Host:           host,
		SingleThreaded: core.TSTrue,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()

	statements := p.GetSourceFile("/main.ts").Statements.Nodes
	importedSymbol := func(index int) *ast.Symbol {
		namedImports := statements[index].AsImportDeclaration().ImportClause.AsImportClause().NamedBindings
		return c.GetAliasedSymbol(c.GetSymbolAtLocation(namedImports.Elements()[0].Name()))
	}
	p1, p2, p3 := importedSymbol(0), importedSymbol(1), importedSymbol(2)
	assert.Assert(t, p1 != p2 && p2 != p3 && p1 != p3, "Expected each import to have its own symbol")

	// The comparison touches p1 and p3 first; p2 is only touched afterwards, through its links.
	assert.Assert(t, c.CompareSymbols(p1, p3) != 0)
	c.GetTypeOfSymbol(p2)
	assert.Assert(t, ast.GetSymbolId(p1) < ast.GetSymbolId(p2), "Expected p1 to have a smaller ID than p2")
	assert.Assert(t, ast.GetSymbolId(p3) < ast.GetSymbolId(p2), "Expected p3 to have a smaller ID than p2")
}

func BenchmarkNewChecker(b *testing.B) {
	fs := bundled.WrapFS(osvfs.FS())
	rootPath := tspath.RootedDirectoryPathFromAbsolute(filepath.Join(repo.TestDataPath(), "fixtures/compiler"))
	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile(rootPath.ResolveFile("tsconfig.json"), &core.CompilerOptions{}, nil, fs, nil)
	assert.Equal(b, len(errors), 0, "Expected no errors in parsed command line")
	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})

	b.ReportAllocs()
	for b.Loop() {
		checker.NewChecker(program, nil)
	}
}

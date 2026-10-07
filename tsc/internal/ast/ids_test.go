package ast_test

import (
	"sync"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"gotest.tools/v3/assert"
)

func TestIdAllocatorPagesAreExclusive(t *testing.T) {
	t.Parallel()

	var allocators [4]ast.IdAllocator
	pages := make(map[ast.SymbolId]int)
	for i := range 100000 {
		a := i % len(allocators)
		symbol := &ast.Symbol{}
		id := allocators[a].GetSymbolId(symbol)
		assert.Equal(t, ast.GetSymbolId(symbol), id)
		page := id / core.LinkStorePageSize
		owner, seen := pages[page]
		if seen {
			assert.Equal(t, owner, a, "page %d holds IDs from two allocators", page)
		} else {
			pages[page] = a
		}
	}
}

func TestIdAllocatorConcurrentUse(t *testing.T) {
	t.Parallel()

	// Both subtests use the same allocators so node and symbol IDs are assigned side by side.
	var shared, own ast.IdAllocator
	t.Run("nodes", func(t *testing.T) {
		t.Parallel()
		testIdAllocatorConcurrentUse(t, &shared, &own, (*ast.IdAllocator).GetNodeId, ast.GetNodeId)
	})
	t.Run("symbols", func(t *testing.T) {
		t.Parallel()
		testIdAllocatorConcurrentUse(t, &shared, &own, (*ast.IdAllocator).GetSymbolId, ast.GetSymbolId)
	})
}

func testIdAllocatorConcurrentUse[T any, Id ~uint64](t *testing.T, shared *ast.IdAllocator, own *ast.IdAllocator, allocate func(*ast.IdAllocator, *T) Id, plain func(*T) Id) {
	const count = 50000

	// The first two minters share an allocator, which must still never hand out an ID twice.
	minters := []struct {
		owner int
		mint  func(*T) Id
	}{
		{0, func(entity *T) Id { return allocate(shared, entity) }},
		{0, func(entity *T) Id { return allocate(shared, entity) }},
		{1, func(entity *T) Id { return allocate(own, entity) }},
		{2, plain},
	}

	contended := make([]T, count)
	privateIds := make([][]Id, len(minters))
	contendedIds := make([][]Id, len(minters))
	var wg sync.WaitGroup
	for i, minter := range minters {
		privateIds[i] = make([]Id, count)
		contendedIds[i] = make([]Id, count)
		wg.Go(func() {
			for j := range count {
				privateIds[i][j] = minter.mint(new(T))
				contendedIds[i][j] = minter.mint(&contended[j])
			}
		})
	}
	wg.Wait()

	seen := make(map[Id]struct{})
	markSeen := func(id Id) {
		if _, ok := seen[id]; ok || id == 0 {
			t.Fatalf("ID %d was handed out twice", id)
		}
		seen[id] = struct{}{}
	}
	pages := make(map[Id]int)
	for i, minter := range minters {
		for _, id := range privateIds[i] {
			markSeen(id)
			page := id / core.LinkStorePageSize
			if owner, ok := pages[page]; ok && owner != minter.owner {
				t.Fatalf("page %d holds IDs from minters %d and %d", page, owner, minter.owner)
			}
			pages[page] = minter.owner
		}
	}
	for j, id := range contendedIds[0] {
		markSeen(id)
		for i := range minters {
			if contendedIds[i][j] != id {
				t.Fatalf("minters 0 and %d disagree on an ID: %d and %d", i, id, contendedIds[i][j])
			}
		}
	}
}

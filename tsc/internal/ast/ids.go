package ast

import (
	"sync/atomic"

	"github.com/microsoft/TypeScript/tsc/internal/core"
)

type (
	NodeId   uint64
	SymbolId uint64
)

// Atomic ids

// Each counter holds the last ID handed out; 0 means unassigned.
var (
	nextNodeId   atomic.Uint64
	nextSymbolId atomic.Uint64
)

func GetNodeId(node *Node) NodeId {
	return NodeId(getId(&node.id, &nextNodeId))
}

func GetSymbolId(symbol *Symbol) SymbolId {
	return SymbolId(getId(&symbol.id, &nextSymbolId))
}

func getId(id *atomic.Uint64, counter *atomic.Uint64) uint64 {
	value := id.Load()
	if value == 0 {
		// Worst case, we burn a few ids if we have to CAS.
		value = counter.Add(1)
		if !id.CompareAndSwap(0, value) {
			value = id.Load()
		}
	}
	return value
}

// Blocks span whole pages so neither another allocator nor the package-level getters can assign IDs within an allocator's page.
const (
	minIdBlockSize = core.LinkStorePageSize
	maxIdBlockSize = 16 * core.LinkStorePageSize
)

// IdAllocator assigns IDs from page-aligned blocks reserved from the global counters; IDs stay unique even under concurrent use.
type IdAllocator struct {
	nodeIds   atomic.Pointer[idBlock]
	symbolIds atomic.Pointer[idBlock]
}

func (a *IdAllocator) GetNodeId(node *Node) NodeId {
	if id := node.id.Load(); id != 0 {
		return NodeId(id)
	}
	return NodeId(assignId(&a.nodeIds, &node.id, &nextNodeId))
}

func (a *IdAllocator) GetSymbolId(symbol *Symbol) SymbolId {
	if id := symbol.id.Load(); id != 0 {
		return SymbolId(id)
	}
	return SymbolId(assignId(&a.symbolIds, &symbol.id, &nextSymbolId))
}

type idBlock struct {
	next atomic.Uint64
	end  uint64
	size uint64
}

func assignId(current *atomic.Pointer[idBlock], id *atomic.Uint64, counter *atomic.Uint64) uint64 {
	for {
		block := current.Load()
		if block != nil {
			// Claim the ID before publishing it so no two callers can publish the same one.
			if value := block.next.Add(1) - 1; value < block.end {
				if !id.CompareAndSwap(0, value) {
					return id.Load()
				}
				return value
			}
		}
		current.CompareAndSwap(block, reserveIdBlock(counter, block))
	}
}

// reserveIdBlock claims the next page-aligned block of IDs from counter, doubling the previous block's size up to the maximum.
func reserveIdBlock(counter *atomic.Uint64, previous *idBlock) *idBlock {
	size := uint64(minIdBlockSize)
	if previous != nil {
		size = min(2*previous.size, maxIdBlockSize)
	}
	for {
		last := counter.Load()
		first := (last/core.LinkStorePageSize + 1) * core.LinkStorePageSize
		if counter.CompareAndSwap(last, first+size-1) {
			block := &idBlock{end: first + size, size: size}
			block.next.Store(first)
			return block
		}
	}
}

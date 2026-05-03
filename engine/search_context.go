package engine

import "sync"

type SearchContext struct {
	History        HistoryTable
	Killers        [MAX_MOVES][2]MoveNG
	stateStorage   [MAX_PLY + 1]StateInfo
	stateStack     [MAX_PLY + 1]*StateInfo
	Nodes          int
	reuseAnyTTMove bool
}

var searchContextPool sync.Pool

func newSearchContext() *SearchContext {
	return &SearchContext{}
}

func borrowSearchContext() *SearchContext {
	if ctx, ok := searchContextPool.Get().(*SearchContext); ok {
		ctx.clear()
		return ctx
	}
	return newSearchContext()
}

func releaseSearchContext(ctx *SearchContext) {
	searchContextPool.Put(ctx)
}

func (ctx *SearchContext) clear() {
	ctx.Nodes = 0
	ctx.reuseAnyTTMove = false
	clear(ctx.Killers[:])
	clear(ctx.History[:])
}

func (ctx *SearchContext) state(pos *PositionNG) *StateInfo {
	idx := len(pos.St)
	if idx >= 0 && idx < len(ctx.stateStorage) {
		return &ctx.stateStorage[idx]
	}
	return &StateInfo{}
}

func (ctx *SearchContext) copyStateStack(src StateInfoStack) StateInfoStack {
	if len(src) == 0 {
		return ctx.stateStack[:0]
	}
	if len(src) > len(ctx.stateStorage) {
		dst := make(StateInfoStack, len(src))
		for i, srcSt := range src {
			if srcSt == nil {
				continue
			}
			dst[i] = &StateInfo{}
			*dst[i] = *srcSt
		}
		return dst
	}

	for i, srcSt := range src {
		if srcSt == nil {
			ctx.stateStack[i] = nil
			continue
		}
		ctx.stateStorage[i] = *srcSt
		ctx.stateStack[i] = &ctx.stateStorage[i]
	}
	return ctx.stateStack[:len(src)]
}

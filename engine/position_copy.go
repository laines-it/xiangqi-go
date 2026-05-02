package engine

import "sync"

type positionCopyOptions struct {
	preserveHeuristics bool
	resetNodes         bool
}

var positionCopyPool sync.Pool

func copyPosition(pos *PositionNG) *PositionNG {
	copied := &PositionNG{}
	copyPositionInto(copied, pos, positionCopyOptions{})
	return copied
}

func borrowPositionCopy(src *PositionNG) *PositionNG {
	if copied, ok := positionCopyPool.Get().(*PositionNG); ok {
		copyPositionInto(copied, src, positionCopyOptions{})
		return copied
	}

	return copyPosition(src)
}

func releasePositionCopy(pos *PositionNG) {
	positionCopyPool.Put(pos)
}

func copyPositionInto(dst, src *PositionNG, opts positionCopyOptions) {
	copyPositionIntoWithStack(dst, src, opts, dst.St)
}

func copyPositionIntoWithStack(dst, src *PositionNG, opts positionCopyOptions, st StateInfoStack) {
	history := dst.History
	killers := dst.Killers

	*dst = *src

	if opts.preserveHeuristics {
		dst.History = history
		dst.Killers = killers
	}
	if opts.resetNodes {
		dst.Nodes = 0
	}
	dst.St = copyStateInfoStackInto(dst, st, src.St)
}

func copyStateInfoStackInto(pos *PositionNG, dst, src StateInfoStack) StateInfoStack {
	if len(src) == 0 {
		if dst == nil {
			return NewStateInfoStack()
		}
		return dst[:0]
	}

	if cap(dst) < len(src) {
		dst = make(StateInfoStack, len(src))
	} else {
		dst = dst[:len(src)]
	}

	for i, srcSt := range src {
		if srcSt == nil {
			dst[i] = nil
			continue
		}

		if state, ok := pos.stateCopySlot(i); ok {
			*state = *srcSt
			dst[i] = state
			continue
		}

		if dst[i] == nil || (i > 0 && pos.isSearchState(dst[i])) {
			dst[i] = &StateInfo{}
		}
		*dst[i] = *srcSt
	}

	return dst
}

func (pos *PositionNG) stateCopySlot(stackIndex int) (*StateInfo, bool) {
	if stackIndex == 0 {
		return nil, false
	}

	stateIndex := stackIndex - 1
	if stateIndex < 0 || stateIndex >= len(pos.searchStates) || stateIndex == pos.GamePly {
		return nil, false
	}

	return &pos.searchStates[stateIndex], true
}

func (pos *PositionNG) isSearchState(st *StateInfo) bool {
	if st == nil {
		return false
	}
	for i := range pos.searchStates {
		if st == &pos.searchStates[i] {
			return true
		}
	}
	return false
}

package engine

import "sync"

type positionCopyOptions struct {
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

func borrowPositionBranch(src *PositionNG, ctx *SearchContext) *PositionNG {
	if copied, ok := positionCopyPool.Get().(*PositionNG); ok {
		copyPositionBranchInto(copied, src, ctx)
		return copied
	}

	copied := &PositionNG{}
	copyPositionBranchInto(copied, src, ctx)
	return copied
}

func releasePositionCopy(pos *PositionNG) {
	positionCopyPool.Put(pos)
}

func copyPositionBranchInto(dst, src *PositionNG, ctx *SearchContext) {
	*dst = *src
	dst.St = ctx.copyStateStack(src.St)
}

func copyPositionInto(dst, src *PositionNG, opts positionCopyOptions) {
	copyPositionIntoWithStack(dst, src, opts, dst.St)
}

func copyPositionIntoWithStack(dst, src *PositionNG, opts positionCopyOptions, st StateInfoStack) {
	*dst = *src
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

		if dst[i] == nil {
			dst[i] = &StateInfo{}
		}
		*dst[i] = *srcSt
	}

	return dst
}

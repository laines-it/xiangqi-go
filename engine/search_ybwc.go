package engine

import (
	"context"
	"runtime"
	"sync"
)

const ybwcMinSplitDepth uint8 = 4

type ybwcContextKey struct{}
type ybwcHelperContextKey struct{}

type ybwcControl struct {
	workers chan struct{}
}

type searchNode struct {
	alpha Value
	beta  Value
}

type ybwcTTProbe struct {
	move  MoveNG
	score Value
	hit   bool
}

func newYBWCContext() context.Context {
	workers := max(runtime.GOMAXPROCS(0)-1, 0)

	return context.WithValue(context.Background(), ybwcContextKey{}, &ybwcControl{
		workers: make(chan struct{}, workers),
	})
}

func tryAcquireYBWCWorker(ctx context.Context, depth uint8) (func(), bool) {
	if depth < ybwcMinSplitDepth {
		return nil, false
	}

	control, ok := ctx.Value(ybwcContextKey{}).(*ybwcControl)
	if !ok || control == nil {
		return nil, false
	}

	select {
	case control.workers <- struct{}{}:
		return func() {
			<-control.workers
		}, true
	default:
		return nil, false
	}
}

func ybwcHelperContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ybwcHelperContextKey{}, true)
}

func isYBWCHelper(ctx context.Context) bool {
	helper, _ := ctx.Value(ybwcHelperContextKey{}).(bool)
	return helper
}

func ybwcReadHashEntry(key, key2 Key, alpha, beta int16, depth, ply uint8) (int16, MoveNG) {
	return readHashEntry(key, key2, alpha, beta, depth, ply)
}

func ybwcWriteHashEntry(key, key2 Key, score int16, bestMove MoveNG, depth, ply uint8, flag int8) {
	writeHashEntryReuseAnyMove(key, key2, score, bestMove, depth, ply, flag)
}

func ybwcProbeTT(pos *PositionNG, depth uint8, alpha, beta Value) ybwcTTProbe {
	score, move := ybwcReadHashEntry(
		pos.St.Top().key,
		pos.St.Top().key2,
		int16(alpha),
		int16(beta),
		depth,
		uint8(pos.GamePly),
	)
	if !IsOKMove(move) {
		return ybwcTTProbe{move: MOVE_NONE}
	}

	probe := ybwcTTProbe{move: move}
	if score == NO_HASH {
		return probe
	}

	probe.score = Value(score)
	probe.hit = true
	return probe
}

func ybwcEnterNode(ctx context.Context, pos *PositionNG, node searchNode, depth uint8) (searchNode, bool, Value) {
	select {
	case <-ctx.Done():
		return node, true, node.alpha
	default:
	}

	if !isYBWCHelper(ctx) {
		PvLength[pos.GamePly] = pos.GamePly
	}

	if pos.IsDraw() {
		return node, true, VALUE_DRAW
	}

	if pos.GamePly > 0 && pos.IsRepetition() {
		return node, true, -MATERIAL_WEIGHTS[W_CANNON]
	}

	if node.alpha < -Value(MATE_VALUE) {
		node.alpha = -Value(MATE_VALUE)
	}
	if node.beta > Value(MATE_VALUE-1) {
		node.beta = Value(MATE_VALUE - 1)
	}
	if node.alpha >= node.beta {
		return node, true, node.alpha
	}

	return node, false, 0
}

func ybwcSearchChild(ctx context.Context, search *SearchContext, pos *PositionNG, move MoveNG, depth uint8, alpha, beta Value) Value {
	pos.DoMove(move, searchState(search, pos))
	score := -negamaxYBWC(ctx, search, pos, searchNode{-beta, -alpha}, depth-1, true)
	pos.UndoMove(move)

	return score
}

func ybwcSearchSibling(ctx context.Context, search *SearchContext, localPos *PositionNG, move MoveNG, depth uint8, localAlpha, localBeta, localBestScore Value) Value {
	localPos.DoMove(move, searchState(search, localPos))

	score := -negamaxYBWC(ctx, search, localPos, searchNode{-localAlpha - 1, -localAlpha}, depth-1, true)
	if ctx.Err() != nil {
		localPos.UndoMove(move)
		return localBestScore
	}

	if score > localAlpha && score < localBeta {
		score = -negamaxYBWC(ctx, search, localPos, searchNode{-localBeta, -localAlpha}, depth-1, true)
		if ctx.Err() != nil {
			localPos.UndoMove(move)
			return localBestScore
		}
	}

	localPos.UndoMove(move)
	return score
}

func ybwcRecordBestLine(ctx context.Context, search *SearchContext, pos *PositionNG, move MoveNG, depth uint8) {
	if isYBWCHelper(ctx) {
		return
	}

	StorePvMove(move, pos.GamePly)
	if !pos.Capture(move) {
		updateQuietHistory(&search.History, move, pos.SideToMove, depth)
	}
}

func ybwcRecordCutoff(ctx context.Context, search *SearchContext, pos *PositionNG, move MoveNG) {
	if isYBWCHelper(ctx) || pos.Capture(move) {
		return
	}

	recordKillerMove(&search.Killers, pos.GamePly, move)
}

func ybwcStoreBound(pos *PositionNG, depth uint8, score, oldAlpha, beta Value, bestMove MoveNG) {
	flag := TT_EXACT
	if score <= oldAlpha {
		flag = TT_ALPHA
	} else if score >= beta {
		flag = TT_BETA
	}

	ybwcWriteHashEntry(pos.St.Top().key, pos.St.Top().key2, int16(score), bestMove, depth, uint8(pos.GamePly), flag)
}

func ybwcSequentialSearch(ctx context.Context, search *SearchContext, pos *PositionNG, node searchNode, depth uint8, doNullMove bool) Value {
	if ctx.Err() != nil {
		return node.alpha
	}
	if isYBWCHelper(ctx) {
		return negamaxABAbort(search, node.alpha, node.beta, pos, depth, doNullMove, nil, nil, nil)
	}
	return negamaxABAbort(search, node.alpha, node.beta, pos, depth, doNullMove, &PvTable, &PvLength, nil)
}

func ybwcCanSplit(ctx context.Context, pos *PositionNG, depth uint8) bool {
	return depth >= ybwcMinSplitDepth && pos.GamePly == 0 && !isYBWCHelper(ctx)
}

func ybwcSearchParallelSiblings(ctx context.Context, search *SearchContext, pos *PositionNG, mp *MovePicker, depth uint8, node searchNode, bestScore Value, bestMove MoveNG) (searchNode, Value, MoveNG, bool) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	childCtx, cancel := context.WithCancel(ybwcHelperContext(ctx))
	defer cancel()

	updateBest := func(move MoveNG, score Value) {
		mu.Lock()
		defer mu.Unlock()

		if childCtx.Err() != nil || score <= bestScore {
			return
		}

		bestScore = score
		bestMove = move
		if score > node.alpha {
			node.alpha = score
		}
		if score >= node.beta {
			cancel()
		}
	}

	searchWindow := func() (Value, Value, Value) {
		mu.Lock()
		defer mu.Unlock()

		return node.alpha, node.beta, bestScore
	}

	for move := nextLegalOrderedMove(pos, mp); move != MOVE_NONE; move = nextLegalOrderedMove(pos, mp) {
		localAlpha, localBeta, localBestScore := searchWindow()
		localMove := move

		releaseWorker, ok := tryAcquireYBWCWorker(childCtx, depth)
		if !ok {
			score := ybwcSearchSibling(childCtx, search, pos, localMove, depth, localAlpha, localBeta, localBestScore)
			updateBest(localMove, score)
			if childCtx.Err() != nil {
				break
			}
			continue
		}

		localSearch := borrowSearchContext()
		localPos := borrowPositionBranch(pos, localSearch)
		wg.Go(func() {
			defer releaseWorker()
			defer releaseSearchContext(localSearch)
			defer releasePositionCopy(localPos)

			score := ybwcSearchSibling(childCtx, localSearch, localPos, localMove, depth, localAlpha, localBeta, localBestScore)
			updateBest(localMove, score)
		})
	}

	wg.Wait()
	return node, bestScore, bestMove, ctx.Err() != nil
}

func NegamaxYBWCUnordered(ctx context.Context, pos *PositionNG, node searchNode, depth uint8, split bool) Value {
	search := newSearchContext()
	pos.St = search.copyStateStack(pos.St)
	return negamaxYBWC(ctx, search, pos, node, depth, true)
}

func NegamaxYBWC(ctx context.Context, pos *PositionNG, node searchNode, depth uint8, doNullMove bool) Value {
	search := newSearchContext()
	pos.St = search.copyStateStack(pos.St)
	return negamaxYBWC(ctx, search, pos, node, depth, doNullMove)
}

func negamaxYBWC(ctx context.Context, search *SearchContext, pos *PositionNG, node searchNode, depth uint8, doNullMove bool) Value {
	node, done, score := ybwcEnterNode(ctx, pos, node, depth)
	if done {
		return score
	}

	if !ybwcCanSplit(ctx, pos, depth) {
		return ybwcSequentialSearch(ctx, search, pos, node, depth, doNullMove)
	}

	if pos.Checkers().IsNotZero() {
		depth++
	}

	oldAlpha := node.alpha
	probe := ybwcProbeTT(pos, depth, node.alpha, node.beta)
	if probe.hit {
		if !isYBWCHelper(ctx) && IsOKMove(probe.move) && pos.Legal(probe.move) {
			StorePvMove(probe.move, pos.GamePly)
		}
		return probe.score
	}

	mp := orderMovesByHeuristicsForDepth(pos, search, probe.move, depth)
	firstMove := nextLegalOrderedMove(pos, &mp)
	if firstMove == MOVE_NONE {
		return -Value(MATE_VALUE) + Value(pos.GamePly)
	}

	bestScore := ybwcSearchChild(ctx, search, pos, firstMove, depth, node.alpha, node.beta)
	if ctx.Err() != nil {
		return node.alpha
	}

	bestMove := firstMove
	ybwcRecordBestLine(ctx, search, pos, bestMove, depth)

	if bestScore >= node.beta {
		ybwcStoreBound(pos, depth, bestScore, oldAlpha, node.beta, bestMove)
		ybwcRecordCutoff(ctx, search, pos, bestMove)
		return bestScore
	}

	if bestScore > node.alpha {
		node.alpha = bestScore
	}

	var aborted bool
	node, bestScore, bestMove, aborted = ybwcSearchParallelSiblings(ctx, search, pos, &mp, depth, node, bestScore, bestMove)
	if aborted {
		return oldAlpha
	}
	if bestMove != firstMove {
		ybwcRecordBestLine(ctx, search, pos, bestMove, depth)
	}
	if bestScore >= node.beta {
		ybwcRecordCutoff(ctx, search, pos, bestMove)
	}

	ybwcStoreBound(pos, depth, bestScore, oldAlpha, node.beta, bestMove)
	return bestScore
}

func ybwcLegalBestMove(pos *PositionNG, bestMove MoveNG) MoveNG {
	if IsOKMove(bestMove) && pos.Legal(bestMove) {
		return bestMove
	}

	var moves [MAX_MOVES]MoveNG
	size := pos.GenerateLEGAL(moves[:])
	if size > 0 {
		StorePvMove(moves[0], 0)
		return moves[0]
	}
	return bestMove
}

func (pos *PositionNG) SearchPositionYBWC(depth uint8) (bestMove MoveNG, score Value) {
	search := newSearchContext()
	beginRootSearch(search, pos)
	if depth < ybwcMinSplitDepth {
		bestMove, score = pos.searchPositionAB(search, depth)
		return ybwcLegalBestMove(pos, bestMove), score
	}

	ctx := newYBWCContext()
	score = negamaxYBWC(ctx, search, pos, searchNode{-VALUE_INFINITE, VALUE_INFINITE}, depth, false)
	bestMove = PvTable[0]
	return ybwcLegalBestMove(pos, bestMove), score
}

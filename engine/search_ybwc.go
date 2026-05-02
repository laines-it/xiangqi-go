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

var ybwcTTMu sync.Mutex

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

func ybwcReadHashEntry(key, key2 Key, alpha, beta int16, bestMove *MoveNG, depth, ply uint8) (int16, MoveNG) {
	ybwcTTMu.Lock()
	defer ybwcTTMu.Unlock()

	return readHashEntry(key, key2, alpha, beta, bestMove, depth, ply)
}

func ybwcWriteHashEntry(key, key2 Key, score int16, bestMove MoveNG, depth, ply uint8, flag int8) {
	ybwcTTMu.Lock()
	defer ybwcTTMu.Unlock()

	writeHashEntry(key, key2, score, bestMove, depth, ply, flag)
}

func ybwcProbeTT(pos *PositionNG, depth uint8, alpha, beta Value) ybwcTTProbe {
	ybwcTTMu.Lock()
	defer ybwcTTMu.Unlock()

	entry := &TT.Entries[pos.St.Top().key&TT.Mask]
	data, ok := entry.Load()
	if !ok || data.Key != pos.St.Top().key || data.Key2 != pos.St.Top().key2 {
		return ybwcTTProbe{move: MOVE_NONE}
	}

	probe := ybwcTTProbe{move: data.Move}
	if data.Depth < depth {
		return probe
	}

	score := data.Score
	ply := int16(pos.GamePly)
	if score < -MATE_SCORE {
		score += ply
	}
	if score > MATE_SCORE {
		score -= ply
	}

	value := Value(score)
	switch data.Flag {
	case TT_EXACT:
		probe.score = value
		probe.hit = true
	case TT_BETA:
		if value >= beta {
			probe.score = value
			probe.hit = true
		}
	case TT_ALPHA:
		if value <= alpha {
			probe.score = value
			probe.hit = true
		}
	}

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

	if depth == 0 {
		return node, true, pos.Evaluate()
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

func ybwcOrderedLegalMoves(pos *PositionNG, ttMove MoveNG, moves []MoveNG) int {
	mp := orderMovesByHeuristics(pos, ttMove)
	size := 0
	for move := nextLegalOrderedMove(pos, &mp); move != MOVE_NONE; move = nextLegalOrderedMove(pos, &mp) {
		moves[size] = move
		size++
	}
	return size
}

func ybwcSearchChild(ctx context.Context, pos *PositionNG, move MoveNG, depth uint8, alpha, beta Value) Value {
	var st StateInfo
	pos.DoMove(move, &st)
	score := -NegamaxYBWC(ctx, pos, searchNode{-beta, -alpha}, depth-1, true)
	pos.UndoMove(move)

	return score
}

func ybwcSearchSibling(ctx context.Context, localPos *PositionNG, move MoveNG, depth uint8, localAlpha, localBeta, localBestScore Value) Value {
	var st StateInfo
	localPos.DoMove(move, &st)

	score := -NegamaxYBWC(ctx, localPos, searchNode{-localAlpha - 1, -localAlpha}, depth-1, true)
	if ctx.Err() != nil {
		localPos.UndoMove(move)
		return localBestScore
	}

	if score > localAlpha && score < localBeta {
		score = -NegamaxYBWC(ctx, localPos, searchNode{-localBeta, -localAlpha}, depth-1, true)
		if ctx.Err() != nil {
			localPos.UndoMove(move)
			return localBestScore
		}
	}

	localPos.UndoMove(move)
	return score
}

func ybwcRecordBestLine(ctx context.Context, pos *PositionNG, move MoveNG, depth uint8) {
	if isYBWCHelper(ctx) {
		return
	}

	StorePvMove(move, pos.GamePly)
	if !pos.Capture(move) {
		mFrom := FromSQ(move)
		mTo := ToSQ(move)
		pos.History[pos.SideToMove][mFrom][mTo] += int32(depth)
	}
}

func ybwcRecordCutoff(ctx context.Context, pos *PositionNG, move MoveNG) {
	if isYBWCHelper(ctx) || pos.Capture(move) {
		return
	}

	pos.Killers[pos.GamePly][1] = pos.Killers[pos.GamePly][0]
	pos.Killers[pos.GamePly][0] = move
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

func NegamaxYBWCUnordered(ctx context.Context, pos *PositionNG, node searchNode, depth uint8, split bool) Value {
	return NegamaxYBWC(ctx, pos, node, depth, true)
}

func NegamaxYBWC(ctx context.Context, pos *PositionNG, node searchNode, depth uint8, doNullMove bool) Value {
	node, done, score := ybwcEnterNode(ctx, pos, node, depth)
	if done {
		return score
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

	var moves [MAX_MOVES]MoveNG
	moveCount := ybwcOrderedLegalMoves(pos, probe.move, moves[:])
	if moveCount == 0 {
		return -Value(MATE_VALUE) + Value(pos.GamePly)
	}

	firstMove := moves[0]
	bestScore := ybwcSearchChild(ctx, pos, firstMove, depth, node.alpha, node.beta)
	if ctx.Err() != nil {
		return node.alpha
	}

	bestMove := firstMove
	ybwcRecordBestLine(ctx, pos, bestMove, depth)

	if bestScore >= node.beta {
		ybwcStoreBound(pos, depth, bestScore, oldAlpha, node.beta, bestMove)
		ybwcRecordCutoff(ctx, pos, bestMove)
		return bestScore
	}

	if bestScore > node.alpha {
		node.alpha = bestScore
	}

	if moveCount > 1 && !isYBWCHelper(ctx) {
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

		for i := 1; i < moveCount; i++ {
			move := moves[i]
			localAlpha, localBeta, localBestScore := searchWindow()
			localMove := move

			releaseWorker, ok := tryAcquireYBWCWorker(childCtx, depth)
			if !ok {
				score := ybwcSearchSibling(childCtx, pos, localMove, depth, localAlpha, localBeta, localBestScore)
				updateBest(localMove, score)
				if childCtx.Err() != nil {
					break
				}
				continue
			}

			localPos := borrowPositionCopy(pos)
			wg.Go(func() {
				defer releaseWorker()
				defer releasePositionCopy(localPos)

				score := ybwcSearchSibling(childCtx, localPos, localMove, depth, localAlpha, localBeta, localBestScore)
				updateBest(localMove, score)
			})
		}

		wg.Wait()

		if ctx.Err() != nil {
			return oldAlpha
		}
		if bestMove != firstMove {
			ybwcRecordBestLine(ctx, pos, bestMove, depth)
		}
		if bestScore >= node.beta {
			ybwcRecordCutoff(ctx, pos, bestMove)
		}
	}

	ybwcStoreBound(pos, depth, bestScore, oldAlpha, node.beta, bestMove)
	return bestScore
}

func (pos *PositionNG) SearchPositionYBWC(depth uint8) (bestMove MoveNG, score Value) {
	clearSearch(pos)
	ctx := newYBWCContext()
	score = NegamaxYBWC(ctx, pos, searchNode{-VALUE_INFINITE, VALUE_INFINITE}, depth, false)
	bestMove = PvTable[0]
	if !IsOKMove(bestMove) || !pos.Legal(bestMove) {
		var moves [MAX_MOVES]MoveNG
		size := pos.GenerateLEGAL(moves[:])
		if size > 0 {
			bestMove = moves[0]
			StorePvMove(bestMove, 0)
		}
	}
	return bestMove, score
}

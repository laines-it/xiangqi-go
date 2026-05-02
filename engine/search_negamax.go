package engine

import "sync"

func negamaxWithContext(ctx *SearchContext, depth uint8, pos *PositionNG, move MoveNG, currentBestScore Value, ordered bool) (bestMove MoveNG, bestScore Value) {
	bestMove = MOVE_NONE
	bestScore = currentBestScore

	pos.DoMove(move, searchState(ctx, pos))
	score := -NegamaxWithContext(ctx, depth-1, pos, ordered)
	pos.UndoMove(move)

	if score > currentBestScore {
		bestScore = score
		bestMove = move
		StorePvMove(move, pos.GamePly)

		// store history moves
		if !pos.Capture(move) {
			mFrom := FromSQ(move)
			mTo := ToSQ(move)
			ctx.History[pos.SideToMove][mFrom][mTo] += int32(depth)
		}
	}

	return bestMove, bestScore
}

func Negamax(depth uint8, pos *PositionNG, ordered bool) (bestScore Value) {
	ctx := newSearchContext()
	pos.St = ctx.copyStateStack(pos.St)
	return NegamaxWithContext(ctx, depth, pos, ordered)
}

func NegamaxWithContext(ctx *SearchContext, depth uint8, pos *PositionNG, ordered bool) (bestScore Value) {
	PvLength[pos.GamePly] = pos.GamePly

	if pos.IsDraw() {
		return 0
	}

	var bestMove MoveNG
	if pos.GamePly > 0 {
		var scoreInt16 int16
		scoreInt16, _ = readHashEntry(pos.St.Top().key, pos.St.Top().key2, 0, 0, &bestMove, depth, uint8(pos.GamePly))
		if scoreInt16 != int16(NO_HASH) {
			return int32(scoreInt16)
		}
	}

	if pos.GamePly > 0 && pos.IsRepetition() {
		return -MATERIAL_WEIGHTS[W_CANNON]
	}

	if depth == 0 {
		return QuiescenceWithContext(ctx, pos, ordered)
	}

	inCheck := pos.Checkers().IsNotZero()
	if inCheck {
		depth++
	}

	bestScore = -int32(MATE_VALUE) + int32(pos.GamePly)
	var legalMoves int

	// loop over moves
	if ordered {
		mp := orderMovesByHeuristics(pos, ctx, bestMove)
		for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {
			legalMoves++
			move, score := negamaxWithContext(ctx, depth, pos, currentMove, bestScore, ordered)
			if score > bestScore {
				bestMove = move
				bestScore = score
			}
		}
	} else {
		var moves [MAX_MOVES]MoveNG
		size := pos.GenerateLEGAL(moves[:])
		for _, currentMove := range moves[:size] {
			if !pos.Legal(currentMove) {
				continue
			}

			legalMoves++

			pos.DoMove(currentMove, searchState(ctx, pos))
			score := -NegamaxWithContext(ctx, depth-1, pos, false)
			pos.UndoMove(currentMove)

			if score > bestScore {
				bestScore = score
				bestMove = currentMove
				StorePvMove(currentMove, pos.GamePly)
				// store history moves
				if !pos.Capture(currentMove) {
					mFrom := FromSQ(currentMove)
					mTo := ToSQ(currentMove)
					ctx.History[pos.SideToMove][mFrom][mTo] += int32(depth)
				}
			}
		}
	}

	// checkmate or stalemate
	if legalMoves == 0 {
		if inCheck {
			return -int32(MATE_VALUE) + int32(pos.GamePly) // checkmate
		}
		return 0 // stalemate
	}

	// store hash entry
	writeHashEntry(pos.St.Top().key, pos.St.Top().key2, int16(bestScore), bestMove, depth, uint8(pos.GamePly), TT_EXACT)

	return bestScore
}

func Quiescence(pos *PositionNG, ordered bool) (bestScore Value) {
	ctx := newSearchContext()
	pos.St = ctx.copyStateStack(pos.St)
	return QuiescenceWithContext(ctx, pos, ordered)
}

func QuiescenceWithContext(ctx *SearchContext, pos *PositionNG, ordered bool) (bestScore Value) {
	PvLength[pos.GamePly] = pos.GamePly

	evalation := pos.Evaluate()
	if pos.GamePly >= int(MAX_MOVES) {
		return evalation
	}

	bestScore = evalation

	mp := orderNoisyMovesByHeuristics(pos, ctx)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {
		if !pos.Capture(currentMove) {
			continue
		}

		pos.DoMove(currentMove, searchState(ctx, pos))
		score := -QuiescenceWithContext(ctx, pos, ordered)
		pos.UndoMove(currentMove)

		if score > bestScore {
			bestScore = score
			StorePvMove(currentMove, pos.GamePly)
		}
	}

	return bestScore
}

type SearchResult struct {
	move  MoveNG
	score Value
	pv    []MoveNG // для восстановления главного варианта
}

// ParallelSearch запускает поиск всех корневых ходов параллельно
func (pos *PositionNG) ParallelSearch(depth uint8) (bestMove MoveNG, bestScore Value) {
	ctx := newSearchContext()
	clearSearch(ctx, pos)
	var wg sync.WaitGroup
	var mu sync.Mutex

	bestScore = -int32(MATE_VALUE)
	results := make([]SearchResult, 0)

	mp := orderMovesByHistory(pos, ctx)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {

		wg.Add(1)
		go func(move MoveNG) {
			defer wg.Done()

			localCtx := borrowSearchContext()
			defer releaseSearchContext(localCtx)
			localPos := borrowPositionBranch(pos, localCtx)
			defer releasePositionCopy(localPos)

			localPos.DoMove(move, searchState(localCtx, localPos))

			clearSearch(localCtx, localPos)
			_, score := localPos.searchPositionAB(localCtx, depth-1)

			localPos.UndoMove(move)

			mu.Lock()
			defer mu.Unlock()

			results = append(results, SearchResult{
				move:  move,
				score: score,
			})

			if score > bestScore {
				bestScore = score
				bestMove = move
			}
		}(currentMove)
	}

	return bestMove, bestScore
}

func (pos *PositionNG) DeepCopy() *PositionNG {
	return copyPosition(pos)
}

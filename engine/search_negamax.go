package engine

import "sync"

func negamax(depth uint8, pos *PositionNG, move MoveNG, currentBestScore Value, ordered bool) (bestMove MoveNG, bestScore Value) {
	bestMove = MOVE_NONE
	bestScore = currentBestScore

	var st StateInfo
	pos.DoMove(move, &st)
	score := -Negamax(depth-1, pos, ordered)
	pos.UndoMove(move)

	if score > currentBestScore {
		bestScore = score
		bestMove = move
		StorePvMove(move, pos.GamePly)

		// store history moves
		if !pos.Capture(move) {
			mFrom := FromSQ(move)
			mTo := ToSQ(move)
			pos.History[pos.SideToMove][mFrom][mTo] += int32(depth)
		}
	}

	return bestMove, bestScore
}

func Negamax(depth uint8, pos *PositionNG, ordered bool) (bestScore Value) {
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
		return Quiescence(pos, ordered)
	}

	inCheck := pos.Checkers().IsNotZero()
	if inCheck {
		depth++
	}

	bestScore = -int32(MATE_VALUE) + int32(pos.GamePly)
	var legalMoves int

	// loop over moves
	if ordered {
		mp := orderMovesByHeuristics(pos, bestMove)
		for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {
			legalMoves++
			move, score := negamax(depth, pos, currentMove, bestScore, ordered)
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

			var st StateInfo
			pos.DoMove(currentMove, &st)
			score := -Negamax(depth-1, pos, false)
			pos.UndoMove(currentMove)

			if score > bestScore {
				bestScore = score
				bestMove = currentMove
				StorePvMove(currentMove, pos.GamePly)
				// store history moves
				if !pos.Capture(currentMove) {
					mFrom := FromSQ(currentMove)
					mTo := ToSQ(currentMove)
					pos.History[pos.SideToMove][mFrom][mTo] += int32(depth)
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
	PvLength[pos.GamePly] = pos.GamePly

	evalation := pos.Evaluate()
	if pos.GamePly >= int(MAX_MOVES) {
		return evalation
	}

	bestScore = evalation

	mp := orderNoisyMovesByHeuristics(pos)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {
		if !pos.Capture(currentMove) {
			continue
		}

		var st StateInfo
		pos.DoMove(currentMove, &st)
		score := -Quiescence(pos, ordered)
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
	clearSearch(pos)
	var wg sync.WaitGroup
	var mu sync.Mutex

	bestScore = -int32(MATE_VALUE)
	results := make([]SearchResult, 0)

	mp := orderMovesByHistory(pos)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {

		wg.Add(1)
		go func(move MoveNG) {
			defer wg.Done()

			localPos := borrowPositionCopy(pos)
			defer releasePositionCopy(localPos)

			var st StateInfo
			localPos.DoMove(move, &st)

			_, score := localPos.SearchPosition_ab(depth - 1)

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

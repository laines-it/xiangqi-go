package engine

import "sync"

func Negamax(depth uint8, pos *PositionNG) (bestScore Value) {
	PvLength[pos.GamePly] = pos.GamePly

	if pos.IsDraw() {
		return 0
	}

	var bestMove MoveNG
	if pos.GamePly > 0 {
		var scoreInt16 int16
		scoreInt16, _ = readHashEntry(pos.St.Top().key, 0, 0, &bestMove, depth, uint8(pos.GamePly))
		if scoreInt16 != int16(NO_HASH) {
			return int32(scoreInt16)
		}
	}

	if pos.GamePly > 0 && pos.IsRepetition() {
		return -MATERIAL_WEIGHTS[W_CANNON]
	}

	if depth == 0 {
		return Quiescence(pos)
	}

	inCheck := pos.Checkers().IsNotZero()
	if inCheck {
		depth++
	}

	bestScore = -int32(MATE_VALUE) + int32(pos.GamePly)
	var legalMoves int

	// loop over moves
	var mp MovePicker
	InitalizeMovePicker(&mp, false, MOVE_NONE, MOVE_NONE, MOVE_NONE, &pos.History)

	for currentMove := SelectNextMove(&mp, pos); currentMove != MOVE_NONE; currentMove = SelectNextMove(&mp, pos) {
		if !pos.Legal(currentMove) {
			continue
		}
		legalMoves++

		var st StateInfo
		pos.DoMove(currentMove, &st)
		score := -Negamax(depth-1, pos)
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

	// checkmate or stalemate
	if legalMoves == 0 {
		if inCheck {
			return -int32(MATE_VALUE) + int32(pos.GamePly) // checkmate
		}
		return 0 // stalemate
	}

	// store hash entry
	writeHashEntry(pos.St.Top().key, int16(bestScore), bestMove, depth, uint8(pos.GamePly), TT_EXACT)

	return bestScore
}

func Quiescence(pos *PositionNG) (bestScore Value) {
	PvLength[pos.GamePly] = pos.GamePly

	evalation := pos.Evaluate()
	if pos.GamePly >= int(MAX_MOVES) {
		return evalation
	}

	bestScore = evalation

	var mp MovePicker
	InitalizeMovePicker(&mp, true, MOVE_NONE, MOVE_NONE, MOVE_NONE, &pos.History)

	for currentMove := SelectNextMove(&mp, pos); currentMove != MOVE_NONE; currentMove = SelectNextMove(&mp, pos) {
		if !pos.Legal(currentMove) {
			continue
		}
		if !pos.Capture(currentMove) {
			continue
		}

		var st StateInfo
		pos.DoMove(currentMove, &st)
		score := -Quiescence(pos)
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

	var mp MovePicker
	InitalizeMovePicker(&mp, false, MOVE_NONE, MOVE_NONE, MOVE_NONE, &pos.History)

	for currentMove := SelectNextMove(&mp, pos); currentMove != MOVE_NONE; currentMove = SelectNextMove(&mp, pos) {
		if !pos.Legal(currentMove) {
			continue
		}

		wg.Add(1)
		go func(move MoveNG) {
			defer wg.Done()

			localPos := pos.DeepCopy()

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
	newPos := &PositionNG{}

	*newPos = *pos

	activeSize := pos.GamePly + 1
	if activeSize > 0 && activeSize <= len(pos.St) {
		newPos.St = make(StateInfoStack, activeSize)
		for i := 0; i < activeSize; i++ {
			if pos.St[i] != nil {
				newSt := &StateInfo{}
				*newSt = *pos.St[i]
				newPos.St[i] = newSt
			}
		}
	} else {
		newPos.St = make(StateInfoStack, 0)
	}

	// History — разделяемая, оставляем как есть

	return newPos
}

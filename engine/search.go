package engine

import (
	"context"
	"sync/atomic"
)

var (
	PvTable  = [MAX_MOVES * MAX_MOVES]MoveNG{}
	PvLength = [MAX_MOVES]int{}
)

type searchPV struct {
	table  [MAX_MOVES * MAX_MOVES]MoveNG
	length [MAX_MOVES]int
}

func (pv *searchPV) clear() {
	clear(pv.table[:])
	clear(pv.length[:])
}

func StorePvMove(move MoveNG, searchPly int) {
	storePvMove(move, searchPly, &PvTable, &PvLength)
}

func searchState(pos *PositionNG) *StateInfo {
	if pos.GamePly >= 0 && pos.GamePly < len(pos.searchStates) {
		return &pos.searchStates[pos.GamePly]
	}
	return &StateInfo{}
}

func storePvMove(move MoveNG, searchPly int, pvTable *[MAX_MOVES * MAX_MOVES]MoveNG, pvLength *[MAX_MOVES]int) {
	if pvTable == nil || pvLength == nil {
		return
	}

	pvTable[searchPly*int(MAX_MOVES)+searchPly] = move
	for nextPly := searchPly + 1; nextPly < pvLength[searchPly+1]; nextPly++ {
		pvTable[searchPly*int(MAX_MOVES)+nextPly] = pvTable[(searchPly+1)*int(MAX_MOVES)+nextPly]
	}
	pvLength[searchPly] = pvLength[searchPly+1]
}

func Quiescence_ab(alpha, beta Value, pos *PositionNG) (bestScore Value) {
	return quiescenceABAbort(alpha, beta, pos, &PvTable, &PvLength, nil)
}

func quiescenceAB(alpha, beta Value, pos *PositionNG, pvTable *[MAX_MOVES * MAX_MOVES]MoveNG, pvLength *[MAX_MOVES]int) (bestScore Value) {
	return quiescenceABAbort(alpha, beta, pos, pvTable, pvLength, nil)
}

func quiescenceABAbort(alpha, beta Value, pos *PositionNG, pvTable *[MAX_MOVES * MAX_MOVES]MoveNG, pvLength *[MAX_MOVES]int, abort *atomic.Bool) (bestScore Value) {
	if abort != nil && abort.Load() {
		return alpha
	}

	if pvLength != nil {
		pvLength[pos.GamePly] = pos.GamePly
	}
	evalation := pos.Evaluate()
	if pos.GamePly >= int(MAX_MOVES) {
		return evalation
	}
	if evalation >= beta {
		return evalation
	}
	if evalation > alpha {
		alpha = evalation
	}

	mp := orderNoisyMovesByHeuristics(pos)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {
		if abort != nil && abort.Load() {
			return alpha
		}

		pos.DoMove(currentMove, searchState(pos))
		score := -quiescenceABAbort(-beta, -alpha, pos, pvTable, pvLength, abort)
		/*
			StorePvMove: a:-38, b: -37, score(0)): -37, ply: 4, move: i0h0, pos:
			-Quiescence_ab(37, 38)
			eval = -34*2 = -68
		*/
		pos.UndoMove(currentMove)
		if score > alpha {
			// Update the Principle Variation
			// fmt.Printf("xxx StorePvMove: %s, score: %d, searchPly: %d\n", pos.MoveStr(currentMove), score, pos.GamePly)
			// fmt.Printf("StorePvMove: a:%d, b: %d, score(%d)): %v, ply: %d, move: %s, pos: %s\n",
			// 	alpha, beta, pos.SideToMove, score, pos.GamePly, Move2Str(currentMove), pos.String())
			storePvMove(currentMove, pos.GamePly, pvTable, pvLength)
			alpha = score

			if score >= beta {
				// log.Printf("xxxxxxxxxxxx==============xxx\n")
				// time.Sleep(time.Second * 5)
				return score
			}
		}
	}
	return alpha
}

func QuiescenceYBWC(ctx context.Context, alpha, beta Value, pos *PositionNG) (bestScore Value) {
	select {
	case <-ctx.Done():
		return alpha
	default:
	}

	PvLength[pos.GamePly] = pos.GamePly
	evalation := pos.Evaluate()
	if pos.GamePly >= int(MAX_MOVES) {
		return evalation
	}
	if evalation >= beta {
		return evalation
	}
	if evalation > alpha {
		alpha = evalation
	}

	mp := orderNoisyMovesByHeuristics(pos)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {

		select {
		case <-ctx.Done():
			return alpha
		default:
		}

		pos.DoMove(currentMove, searchState(pos))
		score := -Quiescence_ab(-beta, -alpha, pos)
		/*
			StorePvMove: a:-38, b: -37, score(0)): -37, ply: 4, move: i0h0, pos:
			-Quiescence_ab(37, 38)
			eval = -34*2 = -68
		*/
		pos.UndoMove(currentMove)
		if score > alpha {
			// Update the Principle Variation
			// fmt.Printf("xxx StorePvMove: %s, score: %d, searchPly: %d\n", pos.MoveStr(currentMove), score, pos.GamePly)
			// fmt.Printf("StorePvMove: a:%d, b: %d, score(%d)): %v, ply: %d, move: %s, pos: %s\n",
			// 	alpha, beta, pos.SideToMove, score, pos.GamePly, Move2Str(currentMove), pos.String())
			StorePvMove(currentMove, pos.GamePly)
			alpha = score

			if score >= beta {
				// log.Printf("xxxxxxxxxxxx==============xxx\n")
				// time.Sleep(time.Second * 5)
				return score
			}
		}
	}
	return alpha
}

func Negamax_ab(alpha, beta Value, pos *PositionNG, depth uint8, doNullMove bool) (bestScore Value) {
	return negamaxABAbort(alpha, beta, pos, depth, doNullMove, &PvTable, &PvLength, nil)
}

func negamaxABWithPV(alpha, beta Value, pos *PositionNG, depth uint8, doNullMove bool, pv *searchPV) Value {
	return negamaxABWithPVAbort(alpha, beta, pos, depth, doNullMove, pv, nil)
}

func negamaxABWithPVAbort(alpha, beta Value, pos *PositionNG, depth uint8, doNullMove bool, pv *searchPV, abort *atomic.Bool) Value {
	if pv == nil {
		return negamaxABAbort(alpha, beta, pos, depth, doNullMove, nil, nil, abort)
	}
	return negamaxABAbort(alpha, beta, pos, depth, doNullMove, &pv.table, &pv.length, abort)
}

func negamaxAB(alpha, beta Value, pos *PositionNG, depth uint8, doNullMove bool, pvTable *[MAX_MOVES * MAX_MOVES]MoveNG, pvLength *[MAX_MOVES]int) (bestScore Value) {
	return negamaxABAbort(alpha, beta, pos, depth, doNullMove, pvTable, pvLength, nil)
}

func negamaxABAbort(alpha, beta Value, pos *PositionNG, depth uint8, doNullMove bool, pvTable *[MAX_MOVES * MAX_MOVES]MoveNG, pvLength *[MAX_MOVES]int, abort *atomic.Bool) (bestScore Value) {
	if abort != nil && abort.Load() {
		return alpha
	}

	if pvLength != nil {
		pvLength[pos.GamePly] = pos.GamePly
	}
	rootNode := pos.GamePly == 0
	pvNode := alpha != beta-1
	hashFlag := TT_ALPHA
	var score Value
	var legalMoves int
	futilityPruning := 0
	if pos.IsDraw() {
		return 0
	}
	var ttMove MoveNG
	var bestMove MoveNG
	if pos.GamePly > 0 {
		var scoreInt16 int16
		scoreInt16, ttMove = readHashEntry(pos.St.Top().key, pos.St.Top().key2, int16(alpha), int16(beta), &bestMove, depth, uint8(pos.GamePly))
		score = int32(scoreInt16)
		if score != int32(NO_HASH) && !pvNode {
			return score
		}
	}
	if pos.GamePly > 0 && pos.IsRepetition() {
		return -MATERIAL_WEIGHTS[W_CANNON]
	}
	if depth == 0 {
		return quiescenceABAbort(alpha, beta, pos, pvTable, pvLength, abort)
	}

	// mate distance pruning
	if alpha < -int32(MATE_VALUE) {
		alpha = -int32(MATE_VALUE)
	}
	if beta > int32(MATE_VALUE-1) {
		beta = int32(MATE_VALUE) - 1
	}
	if alpha >= beta {
		return alpha
	}
	inCheck := pos.Checkers().IsNotZero()
	if inCheck {
		depth++
	}

	var staticEval Value
	staticEvalReady := false
	if !pvNode && !inCheck {
		staticEval = pos.Evaluate()
		staticEvalReady = true
		if depth <= 5 && !rootNode && beta > -1000 && alpha < 1000 {
			if staticEval < alpha-int32(depth)*200 { // fail-low
				return staticEval
			}
			if staticEval > beta+int32(depth)*125 { // fali-high
				return staticEval
			}
		}
		if doNullMove {
			if pos.GamePly > 0 && depth > 2 && staticEval >= beta {
				pos.DoNullMove(searchState(pos))
				score = -negamaxABAbort(-beta, -beta+1, pos, depth-1-2, false, pvTable, pvLength, abort)
				pos.UndoNullMove()

				if score >= beta {
					return beta
				}
			}

			// razoring
			score = staticEval + MATERIAL_WEIGHTS[W_PAWN]

			var newScore Value
			if score < beta {
				if depth == 1 {
					newScore = quiescenceABAbort(alpha, beta, pos, pvTable, pvLength, abort)
					if newScore > score {
						return newScore
					}
					return score
				}
			}
			score += MATERIAL_WEIGHTS[W_PAWN]

			if score < beta && depth < 4 {
				newScore = quiescenceABAbort(alpha, beta, pos, pvTable, pvLength, abort)
				if newScore < beta {
					if newScore > score {
						return newScore
					}
					return score
				}
			}
		}

		// futility pruning condition
		if !staticEvalReady {
			staticEval = pos.Evaluate()
			staticEvalReady = true
		}
		futilityMargin := [...]Value{0, MATERIAL_WEIGHTS[W_PAWN], MATERIAL_WEIGHTS[W_KNIGHT], MATERIAL_WEIGHTS[W_CANNON]}
		if depth < 4 && abs(int(alpha)) < int(MATE_SCORE) && staticEval+futilityMargin[depth] <= alpha {
			futilityPruning = 1
		}
	}

	movesSearched := 0
	// loop over moves
	mp := orderMovesByHeuristics(pos, ttMove)
	for currentMove := nextLegalOrderedMove(pos, &mp); currentMove != MOVE_NONE; currentMove = nextLegalOrderedMove(pos, &mp) {
		if abort != nil && abort.Load() {
			return alpha
		}

		legalMoves++

		// futility pruning
		if futilityPruning > 0 && movesSearched > 0 && !pos.Capture(currentMove) && !pos.GivesCheck(currentMove) {
			continue
		}
		pos.DoMove(currentMove, searchState(pos))
		if depth < 5 || movesSearched == 0 {
			score = -negamaxABAbort(-beta, -alpha, pos, depth-1, true, pvTable, pvLength, abort)
		} else {
			// LMR
			if IsOKMove(pos.Killers[pos.GamePly][0]) && IsOKMove(pos.Killers[pos.GamePly][1]) {
				mFrom := FromSQ(currentMove)
				mTo := ToSQ(currentMove)
				k0From := FromSQ(pos.Killers[pos.GamePly][0])
				k0To := ToSQ(pos.Killers[pos.GamePly][0])
				k1From := FromSQ(pos.Killers[pos.GamePly][1])
				k1To := ToSQ(pos.Killers[pos.GamePly][1])
				if !pvNode && movesSearched > 3 && depth > 2 &&
					!inCheck &&
					(mFrom != k0From || mTo != k0To) &&
					(mFrom != k1From || mTo != k1To) &&
					!pos.Capture(currentMove) {
					score = -negamaxABAbort(-alpha-1, -alpha, pos, depth-2, true, pvTable, pvLength, abort)
				} else {
					score = alpha + 1
				}
			} else {
				score = alpha + 1
			}
			// PVS
			if score > alpha {
				score = -negamaxABAbort(-alpha-1, -alpha, pos, depth-1, true, pvTable, pvLength, abort)
				if (score > alpha) && score < beta {
					score = -negamaxABAbort(-beta, -alpha, pos, depth-1, true, pvTable, pvLength, abort)
				}
			}
		}

		pos.UndoMove(currentMove)
		movesSearched++

		if score > alpha {
			hashFlag = TT_EXACT
			bestMove = currentMove
			alpha = score
			// fmt.Printf("vvv StorePvMove: %s, score: %d, depth: %d, alpha: %d, searchPly: %d\n", pos.MoveStr(currentMove), score, depth, alpha, pos.GamePly)
			storePvMove(currentMove, pos.GamePly, pvTable, pvLength)

			// store history moves
			if !pos.Capture(currentMove) {
				mFrom := FromSQ(currentMove)
				mTo := ToSQ(currentMove)
				pos.History[pos.SideToMove][mFrom][mTo] += int32(depth)
			}
			if score >= beta {
				// store hash entry with the score equal to beta
				writeHashEntry(pos.St.Top().key, pos.St.Top().key2, int16(beta), bestMove, depth, uint8(pos.GamePly), TT_BETA)
				// store killer moves
				if !pos.Capture(currentMove) {
					pos.Killers[pos.GamePly][1] = pos.Killers[pos.GamePly][0]
					pos.Killers[pos.GamePly][0] = currentMove
				}
				return beta
			}
		}
	}

	// checkmate or stalemate is a win
	if legalMoves == 0 {
		return -int32(MATE_VALUE) + int32(pos.GamePly)
	}

	// store hash entry with the score equal to alpha
	writeHashEntry(pos.St.Top().key, pos.St.Top().key2, int16(alpha), bestMove, depth, uint8(pos.GamePly), hashFlag)

	return alpha
}

// search position for the best move
func (pos *PositionNG) SearchPosition_ab(depth uint8) (bestMove MoveNG, score Value) {
	clearSearch(pos)
	//now := time.Now()
	var prevScore Value
	// iterative deepening
	for currentDepth := uint8(1); currentDepth <= depth; currentDepth++ {
		alpha := -VALUE_INFINITE
		beta := VALUE_INFINITE
		if currentDepth > 2 {
			window := Value(40 + 15*int(currentDepth))
			alpha = max(prevScore-window, -VALUE_INFINITE)
			beta = min(prevScore+window, VALUE_INFINITE)
		}
		score := Negamax_ab(alpha, beta, pos, currentDepth, true)
		if currentDepth > 2 && (score <= alpha || score >= beta) {
			score = Negamax_ab(-VALUE_INFINITE, VALUE_INFINITE, pos, currentDepth, true)
		}
		prevScore = score

		// fmt.Printf("info score cp %d depth %d nodes %d time %v pv",
		// 	score, currentDepth, pos.Nodes, time.Since(now))
		// for cnt := 0; cnt < PvLength[0]; cnt++ {
		// 	fmt.Printf(" %s", Move2Str(PvTable[cnt]))
		// }
		// fmt.Println()
	}
	bestMove = PvTable[0]
	return bestMove, prevScore
}

// search position for the best move
func (pos *PositionNG) SearchPosition(depth uint8) (bestMove MoveNG) {
	clearSearch(pos)
	//now := time.Now()
	var prevScore Value
	// iterative deepening
	for currentDepth := uint8(1); currentDepth <= depth; currentDepth++ {
		alpha := -VALUE_INFINITE
		beta := VALUE_INFINITE
		if currentDepth > 2 {
			window := Value(40 + 15*int(currentDepth))
			alpha = max(prevScore-window, -VALUE_INFINITE)
			beta = min(prevScore+window, VALUE_INFINITE)
		}
		score := Negamax(currentDepth, pos, true)
		if currentDepth > 2 && (score <= alpha || score >= beta) {
			score = Negamax(currentDepth, pos, true)
		}
		prevScore = score

		// fmt.Printf("info score cp %d depth %d nodes %d time %v pv",
		// 	score, currentDepth, pos.Nodes, time.Since(now))
		// for cnt := 0; cnt < PvLength[0]; cnt++ {
		// 	fmt.Printf(" %s", Move2Str(PvTable[cnt]))
		// }
		// fmt.Println()
	}
	bestMove = PvTable[0]
	return bestMove
}

func clearSearch(pos *PositionNG) {
	pos.GamePly = 0
	pos.Nodes = 0
	clear(PvTable[:])
	clear(PvLength[:])
	clear(pos.Killers[:])
	clear(pos.History[:])
}

const (
	INFINITY   int16 = 32002
	MATE_VALUE int16 = 32000
	MATE_SCORE int16 = 31000
)

const NO_HASH int16 = 32767

func readHashEntry(key, key2 Key, alpha, beta int16, bestMove *MoveNG, depth, ply uint8) (int16, MoveNG) {
	entry := &TT.Entries[key&TT.Mask]
	data, ok := entry.Load()
	if ok && data.Key == key && data.Key2 == key2 {
		if data.Depth >= depth {
			score := data.Score
			if score < -MATE_SCORE {
				score += int16(ply)
			}
			if score > MATE_SCORE {
				score -= int16(ply)
			}
			if data.Flag == TT_EXACT {
				return score, data.Move
			}
			if data.Flag == TT_ALPHA && score <= alpha {
				return alpha, data.Move
			}
			if data.Flag == TT_BETA && score >= beta {
				return beta, data.Move
			}
		}
	}
	return NO_HASH, MOVE_NONE
}

// write hash entry data
func writeHashEntry(key, key2 Key, score int16, bestMove MoveNG, depth, ply uint8, flag int8) {
	entry := &TT.Entries[key&TT.Mask]
	if score < -MATE_SCORE {
		score -= int16(ply)
	}
	if score > MATE_SCORE {
		score += int16(ply)
	}
	entry.Store(ttEntryData{
		Key:   key,
		Key2:  key2,
		Score: score,
		Flag:  flag,
		Depth: depth,
		Move:  bestMove,
		Age:   uint8(age.Load()),
	})
}

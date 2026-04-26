package engine

import (
	"context"
	"sync"
)

type searchNode struct {
	alpha   Value
	beta    Value
	best    MoveNG
	score   Value
	depth   uint8
	pvNode  bool
	inCheck bool
}

func enterNode(ctx context.Context, pos *PositionNG, node searchNode) (Value, Value, bool, Value) {
	if pos.IsDraw() {
		return node.alpha, node.beta, true, 0
	}

	if pos.GamePly > 0 && pos.IsRepetition() {
		return node.alpha, node.beta, true, -MATERIAL_WEIGHTS[W_CANNON]
	}

	if node.depth == 0 {
		return node.alpha, node.beta, true, QuiescenceYBWC(ctx, node.alpha, node.beta, pos)
	}

	if node.alpha < -int32(MATE_VALUE) {
		node.alpha = -int32(MATE_VALUE)
	}
	if node.beta > int32(MATE_VALUE-1) {
		node.beta = int32(MATE_VALUE) - 1
	}
	if node.alpha >= node.beta {
		return node.alpha, node.beta, true, node.alpha
	}

	return node.alpha, node.beta, false, 0
}

func probeTT(pos *PositionNG, node searchNode) (MoveNG, MoveNG, int8) {
	var ttMove MoveNG
	var bestMove MoveNG
	hashFlag := TT_ALPHA

	if pos.GamePly > 0 {
		var scoreInt16 int16
		scoreInt16, ttMove = readHashEntry(pos.St.Top().key, int16(node.alpha), int16(node.beta), &bestMove, node.depth, uint8(pos.GamePly))
		score := int32(scoreInt16)

		if score == int32(NO_HASH) || node.pvNode {
			return ttMove, bestMove, TT_EXACT
		}
	}

	return ttMove, bestMove, hashFlag
}

// returns Pruning, isPruned, score
func applyStaticPruning(
	ctx context.Context,
	pos *PositionNG,
	node searchNode,
	doNullMove bool,
	rootNode bool,
) (int, bool, Value) {

	if node.pvNode || node.inCheck {
		return 0, false, 0
	}

	staticEval := pos.Evaluate()

	if pruned, score := reverseFutilityPruning(staticEval, node.alpha, node.beta, node.depth, rootNode); pruned {
		return 0, true, score
	}

	if doNullMove {
		if pruned, score := nullMovePruning(ctx, pos, staticEval, node.beta, node.depth); pruned {
			return 0, true, score
		}
	}

	if pruned, score := razoring(ctx, pos, staticEval, node.alpha, node.beta, node.depth); pruned {
		return 0, true, score
	}

	futility := futilityPruningFlag(staticEval, node.alpha, node.depth)

	return futility, false, 0
}

func reverseFutilityPruning(
	staticEval Value,
	alpha, beta Value,
	depth uint8,
	rootNode bool,
) (bool, Value) {

	if depth > 5 || rootNode || beta <= -1000 || alpha >= 1000 {
		return false, 0
	}

	if staticEval < alpha-int32(depth)*200 {
		return true, staticEval
	}

	if staticEval > beta+int32(depth)*125 {
		return true, staticEval
	}

	return false, 0
}

func nullMovePruning(
	ctx context.Context,
	pos *PositionNG,
	staticEval Value,
	beta Value,
	depth uint8,
) (bool, Value) {

	if pos.GamePly == 0 || depth <= 2 || staticEval < beta {
		return false, 0
	}

	var st StateInfo
	pos.DoNullMove(&st)

	score := -NegamaxxYBWC(ctx, -beta, -beta+1, pos, depth-3, false)

	pos.UndoNullMove()

	if score >= beta {
		return true, beta
	}

	return false, 0
}

func razoring(
	ctx context.Context,
	pos *PositionNG,
	staticEval Value,
	alpha, beta Value,
	depth uint8,
) (bool, Value) {

	score := staticEval + MATERIAL_WEIGHTS[W_PAWN]

	if score < beta {
		if depth == 1 {
			qScore := QuiescenceYBWC(ctx, alpha, beta, pos)

			if qScore > score {
				return true, qScore
			}
			return true, score
		}
	}

	score += MATERIAL_WEIGHTS[W_PAWN]

	if score < beta && depth < 4 {
		qScore := QuiescenceYBWC(ctx, alpha, beta, pos)

		if qScore < beta {
			if qScore > score {
				return true, qScore
			}
			return true, score
		}
	}

	return false, 0
}

func futilityPruningFlag(
	staticEval Value,
	alpha Value,
	depth uint8,
) int {

	if depth >= 4 || abs(int(alpha)) >= int(MATE_SCORE) {
		return 0
	}

	futilityMargin := [...]Value{
		0,
		MATERIAL_WEIGHTS[W_PAWN],
		MATERIAL_WEIGHTS[W_KNIGHT],
		MATERIAL_WEIGHTS[W_CANNON],
	}

	if staticEval+futilityMargin[depth] <= alpha {
		return 1
	}

	return 0
}

func initMovePicker(pos *PositionNG, ttMove MoveNG) MovePicker {
	var mp MovePicker
	InitalizeMovePicker(&mp, false, ttMove, pos.Killers[pos.GamePly][0], pos.Killers[pos.GamePly][1], &pos.History)
	return mp
}

func searchFirstMove(
	ctx context.Context,
	pos *PositionNG,
	node searchNode,
	mp MovePicker,
	bestMove MoveNG,
	hashFlag int8,
) (Value, MoveNG, int8, bool) {

	movesSearched := 0

	for move := SelectNextMove(&mp, pos); move != MOVE_NONE; move = SelectNextMove(&mp, pos) {
		if !pos.Legal(move) {
			continue
		}

		var st StateInfo
		pos.DoMove(move, &st)

		score := -NegamaxxYBWC(ctx, pos, node, true)

		pos.UndoMove(move)
		movesSearched++

		if score > node.alpha {
			hashFlag = TT_EXACT
			bestMove = move
			node.alpha = score

			StorePvMove(move, pos.GamePly)

			if !pos.Capture(move) {
				mFrom := FromSQ(move)
				mTo := ToSQ(move)
				pos.History[pos.SideToMove][mFrom][mTo] += int32(node.depth)
			}

			if node.alpha >= node.beta {
				writeHashEntry(pos.St.Top().key, int16(node.beta), bestMove, node.depth, uint8(pos.GamePly), TT_BETA)

				if !pos.Capture(move) {
					pos.Killers[pos.GamePly][1] = pos.Killers[pos.GamePly][0]
					pos.Killers[pos.GamePly][0] = move
				}

				return node.beta, bestMove, hashFlag, true
			}
		}

		return node.alpha, bestMove, hashFlag, false
	}

	return node.alpha, bestMove, hashFlag, false
}

func searchRemainingMovesParallel(
	ctx context.Context,
	pos *PositionNG,
	alpha, beta Value,
	depth uint8,
	mp MovePicker,
	bestMove MoveNG,
	hashFlag int,
	futilityPruning int,
) Value {

	type moveResult struct {
		move  MoveNG
		score Value
	}

	resultCh := make(chan moveResult, 256)
	var wg sync.WaitGroup
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	movesSearched := 1

	for move := SelectNextMove(&mp, pos); move != MOVE_NONE; move = SelectNextMove(&mp, pos) {

		if !pos.Legal(move) {
			continue
		}

		if futilityPruning > 0 && movesSearched > 0 && !pos.Capture(move) && !pos.GivesCheck(move) {
			continue
		}

		if ctx.Err() != nil {
			break
		}

		posCopy := pos.DeepCopy()
		var st StateInfo
		posCopy.DoMove(move, &st)

		wg.Add(1)

		go func(m MoveNG, p *PositionNG) {
			defer wg.Done()

			score := -NegamaxYBWC(ctx, -beta, -alpha, p, depth-1, true)

			select {
			case resultCh <- moveResult{m, score}:
			case <-ctx.Done():
			}
		}(move, posCopy)

		movesSearched++
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case res := <-resultCh:
			mu.Lock()
			if res.score > alpha {
				alpha = res.score
				bestMove = res.move

				StorePvMove(res.move, pos.GamePly)

				if alpha >= beta {
					cancel()
					mu.Unlock()
					return beta
				}
			}
			mu.Unlock()

		case <-done:
			return alpha

		case <-ctx.Done():
			return alpha
		}
	}
}

func NegamaxxYBWC(ctx context.Context, pos *PositionNG, node searchNode, doNullMove bool) Value {
	select {
	case <-ctx.Done():
		return node.alpha
	default:
	}

	PvLength[pos.GamePly] = pos.GamePly

	rootNode := pos.GamePly == 0

	node.alpha, node.beta, done, score := enterNode(ctx, pos, node)
	if done {
		return score
	}

	inCheck := pos.Checkers().IsNotZero()
	if inCheck {
		node.depth++
	}

	futilityPruning, pruned, score := applyStaticPruning(ctx, pos, node, doNullMove, rootNode)
	if pruned {
		return score
	}

	ttMove, bestMove, hashFlag := probeTT(pos, node)

	mp := initMovePicker(pos, ttMove)

	node.alpha, bestMove, hashFlag, cutoff := searchFirstMove(ctx, pos, node, mp, bestMove, hashFlag)
	if cutoff {
		return alpha
	}

	alpha = searchRemainingMovesParallel(ctx, pos, alpha, beta, depth, mp, bestMove, hashFlag, futilityPruning)

	writeHashEntry(pos.St.Top().key, int16(alpha), bestMove, depth, uint8(pos.GamePly), hashFlag)

	return alpha
}

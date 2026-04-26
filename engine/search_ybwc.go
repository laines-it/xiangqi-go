package engine

import (
	"context"
	"sync"
)

type searchNode struct {
	alpha Value
	beta  Value
}

type ybwcResult struct {
	alpha     Value
	bestMove  MoveNG
	hashFlag  int8
	legalMove bool
	cutoff    bool
}

func enterNode(ctx context.Context, pos *PositionNG, node searchNode, depth uint8) (Value, Value, bool, Value) {
	if pos.IsDraw() {
		return node.alpha, node.beta, true, 0
	}

	if pos.GamePly > 0 && pos.IsRepetition() {
		return node.alpha, node.beta, true, -MATERIAL_WEIGHTS[W_CANNON]
	}

	if depth == 0 {
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

func probeTT(pos *PositionNG, node searchNode, depth uint8, pvNode bool) (MoveNG, MoveNG, int8) {
	var ttMove MoveNG
	var bestMove MoveNG
	hashFlag := TT_ALPHA

	if pos.GamePly > 0 {
		var scoreInt16 int16
		scoreInt16, ttMove = readHashEntry(pos.St.Top().key, int16(node.alpha), int16(node.beta), &bestMove, depth, uint8(pos.GamePly))
		score := int32(scoreInt16)

		if score == int32(NO_HASH) || pvNode {
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
	depth uint8,
	pvNode bool,
	inCheck bool,
	doNullMove bool,
	rootNode bool,
) (int, bool, Value) {

	if pvNode || inCheck {
		return 0, false, 0
	}

	staticEval := pos.Evaluate()

	if pruned, score := reverseFutilityPruning(staticEval, node.alpha, node.beta, depth, rootNode); pruned {
		return 0, true, score
	}

	if doNullMove {
		if pruned, score := nullMovePruning(ctx, pos, staticEval, node.beta, depth); pruned {
			return 0, true, score
		}
	}

	if pruned, score := razoring(ctx, pos, staticEval, node.alpha, node.beta, depth); pruned {
		return 0, true, score
	}

	futility := futilityPruningFlag(staticEval, node.alpha, depth)

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

	score := -NegamaxYBWC(ctx, pos, searchNode{-beta, -beta + 1}, depth-3, false)

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

func orderMovesByHeuristics(pos *PositionNG, ttMove MoveNG) MovePicker {
	var mp MovePicker
	InitalizeMovePicker(&mp, false, ttMove, pos.Killers[pos.GamePly][0], pos.Killers[pos.GamePly][1], &pos.History)
	return mp
}

func nextLegalOrderedMove(pos *PositionNG, mp *MovePicker) MoveNG {
	for move := SelectNextMove(mp, pos); move != MOVE_NONE; move = SelectNextMove(mp, pos) {
		if pos.Legal(move) {
			return move
		}
	}

	return MOVE_NONE
}

func updateBestLine(pos *PositionNG, move MoveNG, score Value, depth uint8, result *ybwcResult) {
	result.hashFlag = TT_EXACT
	result.bestMove = move
	result.alpha = score

	StorePvMove(move, pos.GamePly)

	if !pos.Capture(move) {
		mFrom := FromSQ(move)
		mTo := ToSQ(move)
		pos.History[pos.SideToMove][mFrom][mTo] += int32(depth)
	}
}

func storeBetaCutoff(pos *PositionNG, move MoveNG, depth uint8, beta Value, bestMove MoveNG) {
	writeHashEntry(pos.St.Top().key, int16(beta), bestMove, depth, uint8(pos.GamePly), TT_BETA)

	if !pos.Capture(move) {
		pos.Killers[pos.GamePly][1] = pos.Killers[pos.GamePly][0]
		pos.Killers[pos.GamePly][0] = move
	}
}

// The first ordered child is searched synchronously. Because every descendant
// repeats this rule, this worker follows the principal variation before the
// current node is split.
func searchPrincipalVariation(
	ctx context.Context,
	pos *PositionNG,
	node searchNode,
	depth uint8,
	mp *MovePicker,
	bestMove MoveNG,
	hashFlag int8,
) ybwcResult {

	result := ybwcResult{
		alpha:     node.alpha,
		bestMove:  bestMove,
		hashFlag:  hashFlag,
		legalMove: false,
	}

	move := nextLegalOrderedMove(pos, mp)
	if move == MOVE_NONE {
		return result
	}

	result.legalMove = true

	var st StateInfo
	pos.DoMove(move, &st)

	score := -NegamaxYBWC(ctx, pos, searchNode{-node.beta, -node.alpha}, depth-1, true)

	pos.UndoMove(move)

	if score > result.alpha {
		updateBestLine(pos, move, score, depth, &result)

		if result.alpha >= node.beta {
			storeBetaCutoff(pos, move, depth, node.beta, result.bestMove)
			result.alpha = node.beta
			result.cutoff = true
		}
	}

	return result
}

// Once the PV child is fully searched, the rest of this node's ordered children
// can be searched by independent workers.
func searchRemainingMovesParallel(
	ctx context.Context,
	pos *PositionNG,
	alpha, beta Value,
	depth uint8,
	mp *MovePicker,
	bestMove MoveNG,
	hashFlag int8,
	futilityPruning int,
) ybwcResult {

	type moveResult struct {
		move  MoveNG
		score Value
	}

	resultCh := make(chan moveResult, 256)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	movesSearched := 1
	result := ybwcResult{
		alpha:     alpha,
		bestMove:  bestMove,
		hashFlag:  hashFlag,
		legalMove: false,
	}

	for move := nextLegalOrderedMove(pos, mp); move != MOVE_NONE; move = nextLegalOrderedMove(pos, mp) {
		result.legalMove = true

		if futilityPruning > 0 && movesSearched > 0 && !pos.Capture(move) && !pos.GivesCheck(move) {
			movesSearched++
			continue
		}

		if ctx.Err() != nil {
			break
		}

		posCopy := pos.DeepCopy()
		var st StateInfo
		posCopy.DoMove(move, &st)

		wg.Add(1)

		childNode := searchNode{-beta, -result.alpha}

		go func(m MoveNG, p *PositionNG, child searchNode) {
			defer wg.Done()

			score := -NegamaxYBWC(ctx, p, child, depth-1, true)

			select {
			case resultCh <- moveResult{m, score}:
			case <-ctx.Done():
			}
		}(move, posCopy, childNode)

		movesSearched++
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for {
		select {
		case res, ok := <-resultCh:
			if !ok {
				return result
			}

			if res.score > result.alpha {
				updateBestLine(pos, res.move, res.score, depth, &result)

				if result.alpha >= beta {
					storeBetaCutoff(pos, res.move, depth, beta, result.bestMove)
					cancel()
					result.alpha = beta
					result.cutoff = true
					return result
				}
			}

		case <-ctx.Done():
			return result
		}
	}
}

func NegamaxYBWC(ctx context.Context, pos *PositionNG, node searchNode, depth uint8, doNullMove bool) Value {
	select {
	case <-ctx.Done():
		return node.alpha
	default:
	}

	PvLength[pos.GamePly] = pos.GamePly

	rootNode := pos.GamePly == 0
	pvNode := node.alpha != node.beta-1

	alpha, beta, done, score := enterNode(ctx, pos, node, depth)
	if done {
		return score
	}

	node.alpha = alpha
	node.beta = beta

	inCheck := pos.Checkers().IsNotZero()
	if inCheck {
		depth++
	}

	futilityPruning, pruned, score := applyStaticPruning(ctx, pos, node, depth, pvNode, inCheck, doNullMove, rootNode)
	if pruned {
		return score
	}

	ttMove, bestMove, hashFlag := probeTT(pos, node, depth, pvNode)

	mp := orderMovesByHeuristics(pos, ttMove)

	result := searchPrincipalVariation(ctx, pos, node, depth, &mp, bestMove, hashFlag)
	if result.cutoff {
		return result.alpha
	}

	node.alpha = result.alpha
	bestMove = result.bestMove
	hashFlag = result.hashFlag

	siblingResult := searchRemainingMovesParallel(ctx, pos, node.alpha, node.beta, depth, &mp, bestMove, hashFlag, futilityPruning)
	if siblingResult.cutoff {
		return siblingResult.alpha
	}

	if siblingResult.legalMove {
		result.legalMove = true
	}

	node.alpha = siblingResult.alpha
	bestMove = siblingResult.bestMove
	hashFlag = siblingResult.hashFlag

	if !result.legalMove {
		return -int32(MATE_VALUE) + int32(pos.GamePly)
	}

	writeHashEntry(pos.St.Top().key, int16(node.alpha), bestMove, depth, uint8(pos.GamePly), hashFlag)

	return node.alpha
}

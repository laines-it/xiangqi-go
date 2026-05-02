package engine

type moveOrderingHints struct {
	skipQuiets bool
	tableMove  MoveNG
	killer1    MoveNG
	killer2    MoveNG
}

func newMoveOrdering(ctx *SearchContext, hints moveOrderingHints) MovePicker {
	var mp MovePicker
	InitalizeMovePicker(&mp, hints.skipQuiets, hints.tableMove, hints.killer1, hints.killer2, &ctx.History)
	return mp
}

func orderMovesByHeuristics(pos *PositionNG, ctx *SearchContext, ttMove MoveNG) MovePicker {
	return newMoveOrdering(ctx, moveOrderingHints{
		tableMove: ttMove,
		killer1:   ctx.Killers[pos.GamePly][0],
		killer2:   ctx.Killers[pos.GamePly][1],
	})
}

func orderMovesByHistory(ctx *SearchContext) MovePicker {
	return newMoveOrdering(ctx, moveOrderingHints{})
}

func orderNoisyMovesByHeuristics(ctx *SearchContext) MovePicker {
	return newMoveOrdering(ctx, moveOrderingHints{
		skipQuiets: true,
	})
}

// nextLegalOrderedMove возвращает следующий легальный ход из MovePicker, который упорядочен по эвристике. Если ходов больше нет, возвращает MOVE_NONE.
func nextLegalOrderedMove(pos *PositionNG, mp *MovePicker) MoveNG {
	for move := SelectNextMove(mp, pos); move != MOVE_NONE; move = SelectNextMove(mp, pos) {
		if pos.Legal(move) {
			return move
		}
	}

	return MOVE_NONE
}

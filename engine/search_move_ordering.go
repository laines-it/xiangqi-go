package engine

type moveOrderingHints struct {
	skipQuiets bool
	tableMove  MoveNG
	killer1    MoveNG
	killer2    MoveNG
}

func newMoveOrdering(pos *PositionNG, hints moveOrderingHints) MovePicker {
	var mp MovePicker
	InitalizeMovePicker(&mp, hints.skipQuiets, hints.tableMove, hints.killer1, hints.killer2, &pos.History)
	return mp
}

func orderMovesByHeuristics(pos *PositionNG, ttMove MoveNG) MovePicker {
	return newMoveOrdering(pos, moveOrderingHints{
		tableMove: ttMove,
		killer1:   pos.Killers[pos.GamePly][0],
		killer2:   pos.Killers[pos.GamePly][1],
	})
}

func orderMovesByHistory(pos *PositionNG) MovePicker {
	return newMoveOrdering(pos, moveOrderingHints{})
}

func orderNoisyMovesByHeuristics(pos *PositionNG) MovePicker {
	return newMoveOrdering(pos, moveOrderingHints{
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

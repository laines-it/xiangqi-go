package engine

const searchedQuietHistoryCapacity = 32

func historyBonus(depth uint8) Value {
	d := Value(depth)
	return min(400, d*d)
}

func historyMalus(depth uint8) Value {
	return min(400, 2*historyBonus(depth))
}

func UpdateHistory(history *HistoryTable, move MoveNG, color Color, delta Value) {
	from := FromSQ(move)
	to := ToSQ(move)
	entry := history[color][from][to]

	// Ensure the update value is within [-400, 400]
	delta = max(-400, min(400, delta))

	// Ensure the new value is within [-16384, 16384]
	history[color][from][to] += 32*delta - entry*(Value(abs(int(delta))))/512
}

func GetHistoryScore(history *HistoryTable, move MoveNG, color Color) Value {
	return history[color][FromSQ(move)][ToSQ(move)]
}

func updateQuietHistory(history *HistoryTable, move MoveNG, color Color, depth uint8) {
	UpdateHistory(history, move, color, historyBonus(depth))
}

func updateBestQuietHistory(history *HistoryTable, bestMove MoveNG, searched []MoveNG, color Color, depth uint8) {
	updateQuietHistory(history, bestMove, color, depth)
	penalizeQuietHistory(history, searched, color, depth)
}

func penalizeQuietHistory(history *HistoryTable, moves []MoveNG, color Color, depth uint8) {
	malus := historyMalus(depth)
	for _, move := range moves {
		UpdateHistory(history, move, color, -malus)
	}
}

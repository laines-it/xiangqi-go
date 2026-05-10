package engine

import (
	"fmt"
	"testing"
)

// Debug test to understand the invalid move issue
func TestGenerateBenchmarkPositionDebug(t *testing.T) {
	pos := generateBenchmarkPosition(sequentialSearchBenchmarkFEN)

	fmt.Printf("Initial GamePly after generateBenchmarkPosition: %d\n", pos.GamePly)
	fmt.Printf("Initial position legal moves count: %d\n", countLegalMoves(&pos))

	var st StateInfo
	for i := 0; i < 5; i++ {
		fmt.Printf("\n--- Search iteration %d ---\n", i+1)
		fmt.Printf("Before search - GamePly: %d\n", pos.GamePly)
		fmt.Printf("Legal moves: %d\n", countLegalMoves(&pos))

		ctx := newSearchContext()
		fmt.Printf("Before clearSearch - GamePly: %d\n", pos.GamePly)
		clearSearch(ctx, &pos)
		fmt.Printf("After clearSearch - GamePly: %d\n", pos.GamePly)

		move, score := pos.SearchPosition_ab(4)
		fmt.Printf("Move returned: %d (IsOKMove: %v), Score: %d\n", move, IsOKMove(move), score)

		if !IsOKMove(move) && score >= -VALUE_MATE && score <= VALUE_MATE {
			fmt.Printf("ERROR: Invalid move but valid score!\n")
			fmt.Printf("Move: %d, Score: %d\n", move, score)
			fmt.Printf("Side to move: %v, Checkers: %d\n", pos.SideToMove, pos.Checkers())
			break
		}

		if IsOKMove(move) {
			fmt.Printf("Making move %d...\n", move)
			pos.DoMove(move, &st)
			fmt.Printf("After DoMove - GamePly: %d\n", pos.GamePly)
		}
	}
}

func countLegalMoves(pos *PositionNG) uint8 {
	var moves [MAX_MOVES]MoveNG
	size := pos.GenerateLEGAL(moves[:])
	return size
}

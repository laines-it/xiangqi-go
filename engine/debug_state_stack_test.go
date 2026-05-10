package engine

import (
	"fmt"
	"testing"
)

// Debug test to understand state stack behavior
func TestStateStackDebug(t *testing.T) {
	pos := generateBenchmarkPosition(sequentialSearchBenchmarkFEN)

	fmt.Printf("After generateBenchmarkPosition:\n")
	fmt.Printf("  GamePly: %d\n", pos.GamePly)
	fmt.Printf("  State stack length: %d\n", len(pos.St))
	fmt.Printf("  Side to move: %d\n", pos.SideToMove)

	var st StateInfo

	// First search-move cycle
	fmt.Printf("\n--- First search-move cycle ---\n")
	move1, _ := pos.SearchPosition_ab(4)
	fmt.Printf("After first search:\n")
	fmt.Printf("  GamePly: %d\n", pos.GamePly)
	fmt.Printf("  State stack length: %d\n", len(pos.St))

	fmt.Printf("Making move %d...\n", move1)
	pos.DoMove(move1, &st)
	fmt.Printf("After DoMove:\n")
	fmt.Printf("  GamePly: %d\n", pos.GamePly)
	fmt.Printf("  State stack length: %d\n", len(pos.St))
	fmt.Printf("  Side to move: %d\n", pos.SideToMove)

	// Second search
	fmt.Printf("\n--- Second search ---\n")
	fmt.Printf("Before second search:\n")
	fmt.Printf("  GamePly: %d\n", pos.GamePly)
	fmt.Printf("  State stack length: %d\n", len(pos.St))

	ctx := newSearchContext()
	fmt.Printf("Before clearSearch:\n")
	fmt.Printf("  GamePly: %d\n", pos.GamePly)
	fmt.Printf("  State stack length: %d\n", len(pos.St))
	fmt.Printf("  pos.St[0] key: %x\n", pos.St[0].key)

	clearSearch(ctx, &pos)
	fmt.Printf("After clearSearch:\n")
	fmt.Printf("  GamePly: %d\n", pos.GamePly)
	fmt.Printf("  State stack length: %d\n", len(pos.St))
	if len(pos.St) > 0 {
		fmt.Printf("  pos.St[0] key: %x\n", pos.St[0].key)
	}
}

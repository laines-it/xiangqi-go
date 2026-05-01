package engine

import (
	"fmt"
	"testing"
)

const initialFen = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

func TestPositionSetResetsState(t *testing.T) {
	var pos PositionNG
	pos.Set(initialFen)

	moves := []string{"h2e2", "h9g7", "h0g2"}
	var states [3]StateInfo
	for i, moveStr := range moves {
		move, err := ParseUCIMove(&pos, moveStr)
		if err != nil {
			t.Fatalf("parse move %s: %v", moveStr, err)
		}
		pos.DoMove(move, &states[i])
	}

	if !pos.PosIsOk() {
		t.Fatal("position invalid after applying moves")
	}

	pos.Set(initialFen)

	if !pos.PosIsOk() {
		t.Fatal("position invalid after reset")
	}
	if pos.SideToMove != WHITE {
		t.Fatalf("expected white to move after reset, got %d", pos.SideToMove)
	}

	if _, err := ParseUCIMove(&pos, moves[0]); err != nil {
		t.Fatalf("move should be legal after reset: %v", err)
	}
}

func TestZobristSecondaryKeyUpdatesWithMoves(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		move string
	}{
		{
			name: "quiet move",
			fen:  initialFen,
			move: "h2e2",
		},
		{
			name: "capture",
			fen:  "4k4/9/9/4p4/9/9/4R4/9/9/4K4 w - - 0 1",
			move: "e3e6",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pos PositionNG
			pos.Set(tc.fen)

			move, err := ParseUCIMove(&pos, tc.move)
			if err != nil {
				t.Fatalf("parse move %s: %v", tc.move, err)
			}

			oldKey := pos.St.Top().key
			oldKey2 := pos.St.Top().key2
			from := FromSQ(move)
			to := ToSQ(move)
			pc := pos.PieceOn(from)
			captured := pos.PieceOn(to)

			expectedKey := oldKey ^ zkey.side ^ zkey.psq[pc][from] ^ zkey.psq[pc][to]
			expectedKey2 := oldKey2 ^ zkey.side2 ^ zkey.psq2[pc][from] ^ zkey.psq2[pc][to]
			if captured != NO_PIECE {
				expectedKey ^= zkey.psq[captured][to]
				expectedKey2 ^= zkey.psq2[captured][to]
			}

			var st StateInfo
			pos.DoMove(move, &st)

			if pos.St.Top().key != expectedKey {
				t.Fatalf("primary key mismatch: got %d, want %d", pos.St.Top().key, expectedKey)
			}
			if pos.St.Top().key2 != expectedKey2 {
				t.Fatalf("secondary key mismatch: got %d, want %d", pos.St.Top().key2, expectedKey2)
			}

			pos.UndoMove(move)

			if pos.St.Top().key != oldKey {
				t.Fatalf("primary key was not restored: got %d, want %d", pos.St.Top().key, oldKey)
			}
			if pos.St.Top().key2 != oldKey2 {
				t.Fatalf("secondary key was not restored: got %d, want %d", pos.St.Top().key2, oldKey2)
			}
		})
	}
}

func TestTranspositionTableContentsAfterDepth2Search(t *testing.T) {
	TTClear()
	defer TTClear()

	var pos PositionNG
	pos.Set(initialFen)

	const depth = uint8(3)
	bestMove, score := pos.SearchPosition_ab(depth)
	fmt.Printf("search depth=%d bestMove=%s score=%d\n", depth, moveString(bestMove), score)

	used := 0
	for i := range TT.Entries {
		entry := &TT.Entries[i]
		data, ok := entry.Load()
		if ok && data.Key != 0 {
			used++
		}
	}
	if used == 0 {
		t.Fatal("expected transposition table to contain entries after depth 2 search")
	}

	fmt.Printf("transposition table used entries: %d\n", used)
	for index := range TT.Entries {
		entry := &TT.Entries[index]
		data, ok := entry.Load()
		if !ok || data.Key == 0 {
			continue
		}
		fmt.Printf(
			"tt[%d]: key=%d key2=%d depth=%d flag=%s score=%d move=%s age=%d\n",
			index,
			data.Key,
			data.Key2,
			data.Depth,
			ttFlagString(data.Flag),
			data.Score,
			moveString(data.Move),
			data.Age,
		)
	}
}

func TestTranspositionTableMaskUsesAllPowerOfTwoEntries(t *testing.T) {
	tt := NewTranTable(1)
	if len(tt.Entries) == 0 {
		t.Fatal("expected non-empty transposition table")
	}
	if len(tt.Entries)&(len(tt.Entries)-1) != 0 {
		t.Fatalf("expected power-of-two table size, got %d", len(tt.Entries))
	}
	if tt.Mask != uint64(len(tt.Entries)-1) {
		t.Fatalf("mask should cover the whole table: got %d, want %d", tt.Mask, len(tt.Entries)-1)
	}
}

func ttFlagString(flag int8) string {
	switch flag {
	case TT_ALPHA:
		return "TT_ALPHA"
	case TT_BETA:
		return "TT_BETA"
	case TT_EXACT:
		return "TT_EXACT"
	default:
		return "UNKNOWN"
	}
}

func moveString(move MoveNG) string {
	if !IsOKMove(move) {
		return "none"
	}
	return Move2Str(move)
}

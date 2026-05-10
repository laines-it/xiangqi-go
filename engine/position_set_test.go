package engine

import (
	"fmt"
	"reflect"
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

func TestZobristIncrementalMatchesFullRecomputeAfterMoveUndo(t *testing.T) {
	var pos PositionNG
	pos.Set(initialFen)
	assertFullHashMatchesState(t, &pos)

	originalKey := pos.St.Top().key
	originalKey2 := pos.St.Top().key2
	originalMinorKey := pos.St.Top().minorKey

	move, err := ParseUCIMove(&pos, "h2e2")
	if err != nil {
		t.Fatalf("parse move: %v", err)
	}

	var st StateInfo
	pos.DoMove(move, &st)
	assertFullHashMatchesState(t, &pos)

	pos.UndoMove(move)
	assertFullHashMatchesState(t, &pos)

	if pos.St.Top().key != originalKey || pos.St.Top().key2 != originalKey2 {
		t.Fatalf("undo did not restore zobrist keys: got (%d,%d), want (%d,%d)",
			pos.St.Top().key, pos.St.Top().key2, originalKey, originalKey2)
	}
	if pos.St.Top().minorKey != originalMinorKey {
		t.Fatalf("undo did not restore minor key: got %d, want %d", pos.St.Top().minorKey, originalMinorKey)
	}
}

func TestDoMoveUndoRestoresPositionIdentity(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		move string
	}{
		{name: "quiet", fen: initialFen, move: "h2e2"},
		{name: "capture", fen: "4k4/9/9/4p4/9/9/4R4/9/9/4K4 w - - 0 1", move: "e3e6"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pos PositionNG
			pos.Set(tc.fen)
			before := capturePositionSnapshot(&pos)

			move, err := ParseUCIMove(&pos, tc.move)
			if err != nil {
				t.Fatalf("parse move: %v", err)
			}

			var st StateInfo
			pos.DoMove(move, &st)
			pos.UndoMove(move)
			after := capturePositionSnapshot(&pos)

			if !reflect.DeepEqual(after, before) {
				t.Fatalf("position snapshot mismatch after do/undo\nbefore=%#v\nafter=%#v", before, after)
			}
		})
	}
}

type positionSnapshot struct {
	Board      [SQUARE_NB]Piece
	ByTypeBB   [PIECE_TYPE_NB]Bitboard
	ByColorBB  [COLOR_NB]Bitboard
	PieceCount [PIECE_NB]int
	KingSQ     [COLOR_NB]Square
	SideToMove Color
	GamePly    int
	Key        Key
	Key2       Key
	MinorKey   Key
	LegalMoves []MoveNG
}

func capturePositionSnapshot(pos *PositionNG) positionSnapshot {
	var moves [MAX_MOVES]MoveNG
	size := pos.GenerateLEGAL(moves[:])
	legalMoves := make([]MoveNG, size)
	copy(legalMoves, moves[:size])

	return positionSnapshot{
		Board:      pos.Board,
		ByTypeBB:   pos.ByTypeBB,
		ByColorBB:  pos.ByColorBB,
		PieceCount: pos.PieceCount,
		KingSQ:     pos.KingSQ,
		SideToMove: pos.SideToMove,
		GamePly:    pos.GamePly,
		Key:        pos.St.Top().key,
		Key2:       pos.St.Top().key2,
		MinorKey:   pos.St.Top().minorKey,
		LegalMoves: legalMoves,
	}
}

func assertFullHashMatchesState(t *testing.T, pos *PositionNG) {
	t.Helper()

	key, key2 := pos.computeFullZobrist()
	if key != pos.St.Top().key {
		t.Fatalf("primary full hash mismatch: got %d, want %d", pos.St.Top().key, key)
	}
	if key2 != pos.St.Top().key2 {
		t.Fatalf("secondary full hash mismatch: got %d, want %d", pos.St.Top().key2, key2)
	}
	if minorKey := pos.computeFullMinorHash(); minorKey != pos.St.Top().minorKey {
		t.Fatalf("minor full hash mismatch: got %d, want %d", pos.St.Top().minorKey, minorKey)
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

func TestTranspositionTableProbeBounds(t *testing.T) {
	tests := []struct {
		name      string
		score     int16
		flag      int8
		alpha     int16
		beta      int16
		depth     uint8
		wantScore int16
		wantHit   bool
	}{
		{
			name:      "exact with sufficient depth",
			score:     42,
			flag:      TT_EXACT,
			alpha:     -100,
			beta:      100,
			depth:     4,
			wantScore: 42,
			wantHit:   true,
		},
		{
			name:      "lower bound applies only above beta",
			score:     70,
			flag:      TT_BETA,
			alpha:     10,
			beta:      50,
			depth:     4,
			wantScore: 50,
			wantHit:   true,
		},
		{
			name:    "lower bound ignored below beta",
			score:   30,
			flag:    TT_BETA,
			alpha:   10,
			beta:    50,
			depth:   4,
			wantHit: false,
		},
		{
			name:      "upper bound applies only below alpha",
			score:     -30,
			flag:      TT_ALPHA,
			alpha:     -10,
			beta:      50,
			depth:     4,
			wantScore: -10,
			wantHit:   true,
		},
		{
			name:    "upper bound ignored above alpha",
			score:   20,
			flag:    TT_ALPHA,
			alpha:   -10,
			beta:    50,
			depth:   4,
			wantHit: false,
		},
		{
			name:    "too shallow entry ignored",
			score:   42,
			flag:    TT_EXACT,
			alpha:   -100,
			beta:    100,
			depth:   6,
			wantHit: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			TTClear()
			defer TTClear()

			const key Key = 0x12345678
			const key2 Key = 0x90abcdef
			move := MakeMove(SQ_A0, SQ_A1)
			writeHashEntry(key, key2, tc.score, move, 5, 0, tc.flag)

			got, gotMove := readHashEntry(key, key2, tc.alpha, tc.beta, tc.depth, 0)
			if tc.wantHit {
				if got != tc.wantScore {
					t.Fatalf("score=%d, want %d", got, tc.wantScore)
				}
				if gotMove != move {
					t.Fatalf("move=%s, want %s", moveString(gotMove), moveString(move))
				}
				return
			}
			if got != NO_HASH {
				t.Fatalf("expected no cutoff, got score %d", got)
			}
			if gotMove != move {
				t.Fatalf("probe should still return stored move for ordering: got %s, want %s", moveString(gotMove), moveString(move))
			}
		})
	}
}

func TestTranspositionTableRejectsWrongChecksum(t *testing.T) {
	TTClear()
	defer TTClear()

	const key Key = 0x1234
	const key2 Key = 0x5678
	writeHashEntry(key, key2, 10, MakeMove(SQ_A0, SQ_A1), 4, 0, TT_EXACT)

	if _, ok := TTProbe(key, key2+1); ok {
		t.Fatal("TTProbe accepted entry with mismatched secondary key")
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

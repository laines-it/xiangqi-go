package engine

import "testing"

func TestMinorHashChangesOnlyForMinorMoves(t *testing.T) {
	var pos PositionNG
	pos.Set(initialFen)

	startMinorKey := pos.MinorHash()

	majorMove, err := ParseUCIMove(&pos, "h0g2")
	if err != nil {
		t.Fatalf("parse major move: %v", err)
	}
	var majorState StateInfo
	pos.DoMove(majorMove, &majorState)
	if pos.MinorHash() != startMinorKey {
		t.Fatalf("major move changed minor hash: got %d, want %d", pos.MinorHash(), startMinorKey)
	}
	pos.UndoMove(majorMove)

	minorMove, err := ParseUCIMove(&pos, "a3a4")
	if err != nil {
		t.Fatalf("parse minor move: %v", err)
	}
	var minorState StateInfo
	pos.DoMove(minorMove, &minorState)
	if pos.MinorHash() == startMinorKey {
		t.Fatal("minor move did not change minor hash")
	}
	pos.UndoMove(minorMove)

	if pos.MinorHash() != startMinorKey {
		t.Fatalf("undo did not restore minor hash: got %d, want %d", pos.MinorHash(), startMinorKey)
	}
}

func TestMinorHashIgnoresMajorConfiguration(t *testing.T) {
	var withMajors PositionNG
	withMajors.Set(initialFen)

	var withoutMajors PositionNG
	withoutMajors.Set("2bakab2/9/9/p1p1p1p1p/9/9/P1P1P1P1P/9/9/2BAKAB2 w - - 0 1")

	if withMajors.MinorHash() != withoutMajors.MinorHash() {
		t.Fatalf("same minor placement should produce same key: got %d and %d", withMajors.MinorHash(), withoutMajors.MinorHash())
	}
}

func TestMinorHashChangesForEachMinorPieceFamily(t *testing.T) {
	tests := []struct {
		name string
		move string
	}{
		{name: "pawn", move: "a3a4"},
		{name: "advisor", move: "d0e1"},
		{name: "elephant", move: "c0e2"},
		{name: "general", move: "e0e1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pos PositionNG
			pos.Set(initialFen)
			start := pos.MinorHash()

			move, err := ParseUCIMove(&pos, tc.move)
			if err != nil {
				t.Fatalf("parse %s: %v", tc.move, err)
			}

			var st StateInfo
			pos.DoMove(move, &st)
			if pos.MinorHash() == start {
				t.Fatalf("%s move %s did not change minor key", tc.name, tc.move)
			}
			if pos.MinorHash() != pos.computeFullMinorHash() {
				t.Fatalf("%s move %s left incremental minor key out of sync", tc.name, tc.move)
			}
		})
	}
}

func TestMinorHashTableCachesMinorEval(t *testing.T) {
	MHTClear()
	defer MHTClear()

	var pos PositionNG
	pos.Set(initialFen)

	checksum := pos.MinorHash()
	phase := pos.MinorPhase()
	score := pos.probeMinorEval()

	entry, ok := ProbeMHT(checksum, phase)
	if !ok {
		t.Fatal("expected MHT hit after probing minor eval")
	}
	if entry.Score != score {
		t.Fatalf("cached score mismatch: got %d, want %d", entry.Score, score)
	}
	if entry.AdvisorNumR != 2 || entry.AdvisorNumB != 2 {
		t.Fatalf("advisor counts mismatch: red=%d black=%d", entry.AdvisorNumR, entry.AdvisorNumB)
	}
	if entry.ElephantNumR != 2 || entry.ElephantNumB != 2 {
		t.Fatalf("elephant counts mismatch: red=%d black=%d", entry.ElephantNumR, entry.ElephantNumB)
	}
	if entry.PawnNumR != 5 || entry.PawnNumB != 5 {
		t.Fatalf("pawn counts mismatch: red=%d black=%d", entry.PawnNumR, entry.PawnNumB)
	}

	majorMove, err := ParseUCIMove(&pos, "h0g2")
	if err != nil {
		t.Fatalf("parse major move: %v", err)
	}
	var st StateInfo
	pos.DoMove(majorMove, &st)
	defer pos.UndoMove(majorMove)

	if pos.MinorHash() != checksum {
		t.Fatal("major move should reuse the same minor configuration")
	}
	if _, ok := ProbeMHT(pos.MinorHash(), pos.MinorPhase()); !ok {
		t.Fatal("expected MHT hit after major-only move")
	}
}

func TestEvaluateUsesMinorPlusDynamic(t *testing.T) {
	MHTClear()
	defer MHTClear()

	var pos PositionNG
	pos.Set(initialFen)

	minor := pos.probeMinorEval()
	dynamic := pos.ComputeDynamicEval()
	want := minor + dynamic
	if pos.SideToMove == BLACK {
		want = -want
	}

	if got := pos.Evaluate(); got != want {
		t.Fatalf("Evaluate()=%d, want minor+dynamic=%d", got, want)
	}
}

func TestMinorHashTableMaskUsesAllPowerOfTwoEntries(t *testing.T) {
	mht := NewMinorHashTable(1)
	if len(mht.Entries) == 0 {
		t.Fatal("expected non-empty minor hash table")
	}
	if len(mht.Entries)&(len(mht.Entries)-1) != 0 {
		t.Fatalf("expected power-of-two MHT size, got %d", len(mht.Entries))
	}
	if mht.Mask != uint64(len(mht.Entries)-1) {
		t.Fatalf("MHT mask should cover the whole table: got %d, want %d", mht.Mask, len(mht.Entries)-1)
	}
}

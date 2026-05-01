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

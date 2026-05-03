package engine

import "testing"

func TestFENRoundTrip(t *testing.T) {
	tests := []string{
		initialFen,
		"1C2ka3/9/C1Nab1n2/p3p3p/6p2/9/P3P3P/3AB4/3p2c2/c1BAK4 w - - 0 1",
		"rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR b - - 12 7",
	}

	for _, fen := range tests {
		t.Run(fen, func(t *testing.T) {
			var pos PositionNG
			pos.Set(fen)
			if got := pos.FEN(); got != fen {
				t.Fatalf("FEN mismatch:\n got: %s\nwant: %s", got, fen)
			}
		})
	}
}

func TestFENAfterMove(t *testing.T) {
	var pos PositionNG
	pos.Set(initialFen)

	move, err := ParseUCIMove(&pos, "h2e2")
	if err != nil {
		t.Fatalf("parse move: %v", err)
	}
	var st StateInfo
	pos.DoMove(move, &st)

	want := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C2C4/9/RNBAKABNR b - - 1 1"
	if got := pos.FEN(); got != want {
		t.Fatalf("FEN mismatch after move:\n got: %s\nwant: %s", got, want)
	}
}

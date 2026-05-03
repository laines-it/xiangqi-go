package pikafish

import (
	"context"
	"testing"
	"time"

	"github.com/hmgle/godogpaw/engine"
)

func TestParseScore(t *testing.T) {
	tests := []struct {
		line  string
		want  int32
		found bool
	}{
		{"info depth 8 score cp 34 nodes 10", 34, true},
		{"info depth 8 score mate 3 nodes 10", int32(engine.VALUE_MATE - 3), true},
		{"info depth 8 score mate -2 nodes 10", int32(-engine.VALUE_MATE + 2), true},
		{"info depth 8 nodes 10", int32(engine.VALUE_DRAW), false},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			got, found := parseScore(tc.line)
			if found != tc.found {
				t.Fatalf("found mismatch: got %v, want %v", found, tc.found)
			}
			if got != tc.want {
				t.Fatalf("score mismatch: got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestLocalPikafishSmoke(t *testing.T) {
	if resolveDefaultPath() == defaultPath {
		t.Skip("local Pikafish binary is not installed")
	}

	client, err := NewFromEnv()
	if err != nil {
		t.Fatalf("initialize Pikafish: %v", err)
	}
	defer client.Close()

	var pos engine.PositionNG
	pos.Set("rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := client.BestMove(ctx, pos.FEN(), 1)
	if err != nil {
		t.Fatalf("search with Pikafish: %v", err)
	}
	move, err := engine.ParseUCIMove(&pos, result.BestMove)
	if err != nil {
		t.Fatalf("parse Pikafish move: %v", err)
	}
	if !engine.IsOKMove(move) {
		t.Fatalf("expected a legal move, got %d", move)
	}
}

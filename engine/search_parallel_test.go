package engine

import "testing"

func TestYBWCMatchesAlphaBetaOnFixedFEN(t *testing.T) {
	fens := []string{
		"4k4/9/9/4p4/9/9/4P4/9/9/4K4 w - - 0 1",
	}
	depths := []uint8{4, 5, 6}

	for _, fen := range fens {
		for _, depth := range depths {
			t.Run(fen+"_depth_"+itoaBenchmark(int(depth)), func(t *testing.T) {
				TTClear()
				MHTClear()
				var alphaBetaPos PositionNG
				alphaBetaPos.Set(fen)
				alphaBetaMove, alphaBetaScore := alphaBetaPos.SearchPosition_ab(depth)

				TTClear()
				MHTClear()
				var ybwcPos PositionNG
				ybwcPos.Set(fen)
				ybwcMove, ybwcScore := ybwcPos.SearchPositionYBWC(depth)

				if ybwcScore != alphaBetaScore && ybwcMove != alphaBetaMove {
					t.Fatalf("YBWC mismatch at depth %d: move=%s score=%d, alpha-beta move=%s score=%d",
						depth, moveString(ybwcMove), ybwcScore, moveString(alphaBetaMove), alphaBetaScore)
				}
			})
		}
	}
}

package engine

import "testing"

func TestTTAgingMakesStaleDeepEntryReplaceable(t *testing.T) {
	stored := ttEntryData{
		Depth: 10,
		Flag:  TT_EXACT,
		Age:   20,
	}

	if shouldReplaceTTEntry(stored, true, 4, TT_BETA, 23) {
		t.Fatal("fresh deep entry should be kept")
	}
	if !shouldReplaceTTEntry(stored, true, 4, TT_BETA, 24) {
		t.Fatal("stale deep entry should be replaced")
	}
}

func TestRootSearchAdvancesTTAgeOnce(t *testing.T) {
	tests := []struct {
		name   string
		search func(*PositionNG)
	}{
		{
			name: "alpha beta",
			search: func(pos *PositionNG) {
				pos.SearchPosition_ab(0)
			},
		},
		{
			name: "negamax",
			search: func(pos *PositionNG) {
				pos.SearchPosition(0)
			},
		},
		{
			name: "lazy smp",
			search: func(pos *PositionNG) {
				pos.SearchPositionLazySMP(0, 2)
			},
		},
		{
			name: "ybwc",
			search: func(pos *PositionNG) {
				pos.SearchPositionYBWC(0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			TTClear()
			defer TTClear()

			var pos PositionNG
			pos.Set(initialFen)
			tc.search(&pos)

			if got := age.Load(); got != 1 {
				t.Fatalf("age=%d, want 1", got)
			}
		})
	}
}

func TestClearSearchDoesNotAdvanceTTAge(t *testing.T) {
	TTClear()
	defer TTClear()

	var pos PositionNG
	pos.Set(initialFen)
	clearSearch(newSearchContext(), &pos)

	if got := age.Load(); got != 0 {
		t.Fatalf("age=%d after internal search reset, want 0", got)
	}
}

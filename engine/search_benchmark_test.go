package engine

import (
	"testing"
	"time"
)

var negamaxBenchmarkScoreSink Value
var negamaxBenchmarkMoveSink MoveNG

var sequentialSearchBenchmarkScoreSink Value
var sequentialSearchBenchmarkMoveSink MoveNG

const (
	sequentialSearchBenchmarkMoves         = 5
	sequentialSearchBenchmarkLazySMPThread = 2
)

var sequentialSearchBenchmarkDepths = []uint8{6, 8, 10}
var sequentialSearchBenchmarkFENs = []string{
	"rnbakabCr/9/7c1/p1p1p1p2/1c6p/9/P1P1P1P1P/9/1C2K4/RNBA1ABNR b - - 0 1",
	"rnbaka1nr/9/3c4b/p1p1p1p1p/7C1/9/P1P1P1P1P/1c7/9/RNBAKABNR w - - 0 1",
	"rCbakab1r/9/6nc1/2p1p3p/p5p2/1c7/P1P1P1PCP/R8/9/1NBAKABNR b - - 0 1",
	"1nbakCbr1/r8/1c4c2/p1p3p1p/4p4/8P/P1P1P1P2/1CN6/8R/R1BAKABN1 w - - 0 1",
	"r1bk1abnr/5c3/9/p1p1pC2p/3C5/9/P1P1P1P2/8B/9/RNBAKA1Nc b - - 0 1",
	"2b2kb2/9/r7r/pc2p1p1p/2p6/9/P1P1c1P1P/9/9/RNBAKABNR w - - 0 1",
	"1nbakabr1/9/r8/p1pC2p1p/9/9/P1P1p1P1P/2N1B1Nc1/4A4/1R1cK1B1R b - - 0 1",
	"rnbakabn1/8C/1r7/p1p3p1p/4p4/2P6/P3P1P1P/7C1/2R6/2B1K1c1R w - - 0 1",
	"1rbaka3/5R3/4b3r/p1p3p1p/4P4/4c4/P1P3P1P/4C4/9/RNBK2B2 b - - 0 1",
	"2b1kabnr/R3aC3/2n6/2p1p1p2/9/4P3p/2P3P2/9/2c1A4/1NB2KB1R w - - 0 1",
}

var negamaxBenchmarkCases = []struct {
	name  string
	fen   string
	depth uint8
}{
	{
		name:  "initial_depth_6",
		fen:   initialFen,
		depth: 4,
	},
	{
		name:  "initial_depth_8",
		fen:   initialFen,
		depth: 8,
	},
	{
		name:  "midgame_depth_8",
		fen:   "1C2ka3/9/C1Nab1n2/p3p3p/6p2/9/P3P3P/3AB4/3p2c2/c1BAK4 w - - 0 1",
		depth: 8,
	},
}

func BenchmarkNegamaxABVsYBWC(b *testing.B) {
	for _, tc := range negamaxBenchmarkCases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.Run("Negamax_ab", func(b *testing.B) {
				benchmarkNegamaxAB(b, tc.fen, tc.depth)
			})
			b.Run("NegamaxYBWC", func(b *testing.B) {
				benchmarkNegamaxYBWC(b, tc.fen, tc.depth)
			})
		})
	}
}

func BenchmarkLazySMPSearchAverage(b *testing.B) {
	threadCounts := []int{1, 2, 4, 6}

	for _, tc := range negamaxBenchmarkCases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			for _, threads := range threadCounts {
				b.Run("threads_"+itoaBenchmark(threads), func(b *testing.B) {
					benchmarkLazySMPSearch(b, tc.fen, tc.depth, threads)
				})
			}
		})
	}
}

func BenchmarkSequentialSearchMoves(b *testing.B) {
	for _, depth := range sequentialSearchBenchmarkDepths {
		b.Run("depth_"+itoaBenchmark(int(depth)), func(b *testing.B) {
			b.Run("AlphaBeta", func(b *testing.B) {
				benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPosition_ab(depth)
				})
			})
			b.Run("YBWC", func(b *testing.B) {
				benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPositionYBWC(depth)
				})
			})
			b.Run("LazySMP", func(b *testing.B) {
				b.Run("threads_"+itoaBenchmark(sequentialSearchBenchmarkLazySMPThread), func(b *testing.B) {
					benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
						return pos.SearchPositionLazySMP(depth, sequentialSearchBenchmarkLazySMPThread)
					})
				})
			})
		})
	}
}

func BenchmarkSearchAccuracyFixedFEN(b *testing.B) {
	for _, depth := range sequentialSearchBenchmarkDepths {
		b.Run("depth_"+itoaBenchmark(int(depth)), func(b *testing.B) {
			b.Run("YBWC", func(b *testing.B) {
				benchmarkSearchAccuracyFixedFEN(b, depth, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPositionYBWC(depth)
				})
			})
			b.Run("LazySMP", func(b *testing.B) {
				b.Run("threads_"+itoaBenchmark(sequentialSearchBenchmarkLazySMPThread), func(b *testing.B) {
					benchmarkSearchAccuracyFixedFEN(b, depth, func(pos *PositionNG) (MoveNG, Value) {
						return pos.SearchPositionLazySMP(depth, sequentialSearchBenchmarkLazySMPThread)
					})
				})
			})
		})
	}
}

func BenchmarkSearchCountersFixedFEN(b *testing.B) {
	for _, depth := range sequentialSearchBenchmarkDepths {
		b.Run("depth_"+itoaBenchmark(int(depth)), func(b *testing.B) {
			b.Run("AlphaBeta", func(b *testing.B) {
				benchmarkSearchCountersFixedFEN(b, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPosition_ab(depth)
				})
			})
			b.Run("YBWC", func(b *testing.B) {
				benchmarkSearchCountersFixedFEN(b, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPositionYBWC(depth)
				})
			})
			b.Run("LazySMP", func(b *testing.B) {
				b.Run("threads_"+itoaBenchmark(sequentialSearchBenchmarkLazySMPThread), func(b *testing.B) {
					benchmarkSearchCountersFixedFEN(b, func(pos *PositionNG) (MoveNG, Value) {
						return pos.SearchPositionLazySMP(depth, sequentialSearchBenchmarkLazySMPThread)
					})
				})
			})
		})
	}
}

func benchmarkSequentialSearchMoves(b *testing.B, search func(*PositionNG) (MoveNG, Value)) {
	b.Helper()
	b.ReportAllocs()
	b.StopTimer()

	var totalSearchNanos int64
	var searchedMoves int
	var searchedSequences int

	for i := 0; i < b.N; i++ {
		for _, fen := range sequentialSearchBenchmarkFENs {
			var pos PositionNG
			pos.Set(fen)
			TTClear()
			MHTClear()

			elapsed := runSequentialSearchBenchmarkMoves(b, &pos, search)

			totalSearchNanos += elapsed.Nanoseconds()
			searchedMoves += sequentialSearchBenchmarkMoves
			searchedSequences++
		}
	}

	if searchedSequences > 0 {
		b.ReportMetric(float64(totalSearchNanos)/float64(searchedSequences)/1e6, "ms/sequence")
	}
	if searchedMoves > 0 {
		b.ReportMetric(float64(totalSearchNanos)/float64(searchedMoves)/1e6, "ms/move")
	}
}

func benchmarkSearchCountersFixedFEN(b *testing.B, search func(*PositionNG) (MoveNG, Value)) {
	b.Helper()
	b.ReportAllocs()
	b.StopTimer()

	var totalPositions uint64
	var totalEvaluates uint64
	var searchedMoves int
	var searchedSequences int

	for i := 0; i < b.N; i++ {
		var total_elapsed time.Duration
		for _, fen := range sequentialSearchBenchmarkFENs {
			var pos PositionNG
			pos.Set(fen)
			TTClear()
			MHTClear()

			visitedPositionCount.Store(0)
			evaluateCallCount.Store(0)

			elapsed := runSequentialSearchBenchmarkMoves(b, &pos, search)
			total_elapsed += elapsed

			totalPositions += visitedPositionCount.Load()
			totalEvaluates += evaluateCallCount.Load()
			searchedMoves += sequentialSearchBenchmarkMoves
			searchedSequences++
		}
		average_time := float64(total_elapsed.Milliseconds()) / float64(searchedSequences*sequentialSearchBenchmarkMoves)
		b.ReportMetric(average_time, "ms/move")
	}

	if searchedSequences > 0 {
		b.ReportMetric(float64(totalPositions)/float64(searchedSequences), "positions/sequence")
		b.ReportMetric(float64(totalEvaluates)/float64(searchedSequences), "evals/sequence")
	}
	if searchedMoves > 0 {
		b.ReportMetric(float64(totalPositions)/float64(searchedMoves), "positions/move")
		b.ReportMetric(float64(totalEvaluates)/float64(searchedMoves), "evals/move")
	}
}

func benchmarkSearchAccuracyFixedFEN(b *testing.B, depth uint8, search func(*PositionNG) (MoveNG, Value)) {
	b.Helper()
	b.ReportAllocs()

	success := 0
	attempts := 0
	for i := 0; i < b.N; i++ {
		for _, fen := range sequentialSearchBenchmarkFENs {
			expectedMove, expectedScore := benchmarkAlphaBetaExpected(fen, depth)
			foundMove, foundScore := benchmarkSearchResult(fen, search)

			if foundMove == expectedMove || foundScore == expectedScore {
				success++
			}
			attempts++
		}
	}

	if attempts > 0 {
		b.ReportMetric(float64(success)/float64(attempts), "accuracy")
	}
}

func benchmarkAlphaBetaExpected(fen string, depth uint8) (MoveNG, Value) {
	var pos PositionNG
	pos.Set(fen)
	TTClear()
	MHTClear()
	return pos.SearchPosition_ab(depth)
}

func benchmarkSearchResult(fen string, search func(*PositionNG) (MoveNG, Value)) (MoveNG, Value) {
	var pos PositionNG
	pos.Set(fen)
	TTClear()
	MHTClear()
	return search(&pos)
}

func runSequentialSearchBenchmarkMoves(b *testing.B, pos *PositionNG, search func(*PositionNG) (MoveNG, Value)) time.Duration {
	b.Helper()

	total_time := time.Duration(0)
	for i := 0; i < sequentialSearchBenchmarkMoves; i++ {

		b.StartTimer()
		start := time.Now()
		move, score := search(pos)
		elapsed := time.Since(start)
		b.StopTimer()

		if !IsOKMove(move) && score >= -VALUE_MATE && score <= VALUE_MATE {
			b.Fatalf("search returned invalid move at move %d", i+1)
		}

		total_time += elapsed

		var st StateInfo
		pos.DoMove(move, &st)

		sequentialSearchBenchmarkMoveSink = move
		sequentialSearchBenchmarkScoreSink = score
	}
	return total_time
}

func benchmarkLazySMPSearch(b *testing.B, fen string, depth uint8, threads int) {
	b.ReportAllocs()
	b.StopTimer()

	var totalNanos int64
	for i := 0; i < b.N; i++ {
		pos := prepareBenchmarkPosition(b, fen)

		b.StartTimer()
		start := time.Now()
		move, score := pos.SearchPositionLazySMP(depth, threads)
		elapsed := time.Since(start)
		b.StopTimer()

		totalNanos += elapsed.Nanoseconds()
		negamaxBenchmarkMoveSink = move
		negamaxBenchmarkScoreSink = score
	}

	if b.N > 0 {
		b.ReportMetric(float64(totalNanos)/float64(b.N)/1e6, "ms/search")
	}
}

func benchmarkNegamaxAB(b *testing.B, fen string, depth uint8) {
	b.ReportAllocs()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		pos := prepareBenchmarkPosition(b, fen)

		b.StartTimer()
		score := Negamax_ab(-VALUE_INFINITE, VALUE_INFINITE, pos, depth, true)
		b.StopTimer()

		negamaxBenchmarkScoreSink = score
		negamaxBenchmarkMoveSink = PvTable[0]
	}
}

func itoaBenchmark(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func benchmarkNegamaxYBWC(b *testing.B, fen string, depth uint8) {
	b.ReportAllocs()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		pos := prepareBenchmarkPosition(b, fen)
		ctx := newYBWCContext()

		b.StartTimer()
		score := NegamaxYBWC(ctx, pos, searchNode{-VALUE_INFINITE, VALUE_INFINITE}, depth, true)
		b.StopTimer()

		negamaxBenchmarkScoreSink = score
		negamaxBenchmarkMoveSink = PvTable[0]
	}
}

func prepareBenchmarkPosition(b *testing.B, fen string) *PositionNG {
	b.Helper()
	b.StopTimer()

	TTClear()
	MHTClear()

	var pos PositionNG
	pos.Set(fen)
	clearSearch(newSearchContext(), &pos)

	return &pos
}

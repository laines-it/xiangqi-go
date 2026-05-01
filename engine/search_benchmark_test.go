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
	sequentialSearchBenchmarkMoves = 5
	sequentialSearchBenchmarkFEN   = "1C2ka3/9/C1Nab1n2/p3p3p/6p2/9/P3P3P/3AB4/3p2c2/c1BAK4 w - - 0 1"
)

var sequentialSearchBenchmarkDepths = []uint8{6, 8, 10}
var sequentialSearchBenchmarkLazySMPThreads = []int{0, 2, 4, 6}

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
				threads := threads
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
				benchmarkSequentialSearchMoves(b, depth, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPosition_ab(depth)
				})
			})
			b.Run("YBWC", func(b *testing.B) {
				benchmarkSequentialSearchMoves(b, depth, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPositionYBWC(depth)
				})
			})
			b.Run("LazySMP", func(b *testing.B) {
				for _, threads := range sequentialSearchBenchmarkLazySMPThreads {
					threads := threads
					b.Run("threads_"+itoaBenchmark(threads), func(b *testing.B) {
						benchmarkSequentialSearchMoves(b, depth, func(pos *PositionNG) (MoveNG, Value) {
							return pos.SearchPositionLazySMP(depth, threads)
						})
					})
				}
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
				for _, threads := range sequentialSearchBenchmarkLazySMPThreads {
					threads := threads
					b.Run("threads_"+itoaBenchmark(threads), func(b *testing.B) {
						benchmarkSearchAccuracyFixedFEN(b, depth, func(pos *PositionNG) (MoveNG, Value) {
							return pos.SearchPositionLazySMP(depth, threads)
						})
					})
				}
			})
		})
	}
}

func BenchmarkSearchCountersFixedFEN(b *testing.B) {
	for _, depth := range sequentialSearchBenchmarkDepths {
		b.Run("depth_"+itoaBenchmark(int(depth)), func(b *testing.B) {
			b.Run("AlphaBeta", func(b *testing.B) {
				benchmarkSearchCountersFixedFEN(b, depth, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPosition_ab(depth)
				})
			})
			b.Run("YBWC", func(b *testing.B) {
				benchmarkSearchCountersFixedFEN(b, depth, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPositionYBWC(depth)
				})
			})
			b.Run("LazySMP", func(b *testing.B) {
				for _, threads := range sequentialSearchBenchmarkLazySMPThreads {
					threads := threads
					b.Run("threads_"+itoaBenchmark(threads), func(b *testing.B) {
						benchmarkSearchCountersFixedFEN(b, depth, func(pos *PositionNG) (MoveNG, Value) {
							return pos.SearchPositionLazySMP(depth, threads)
						})
					})
				}
			})
		})
	}
}

func benchmarkSequentialSearchMoves(b *testing.B, depth uint8, search func(*PositionNG) (MoveNG, Value)) {
	b.Helper()
	b.ReportAllocs()
	b.StopTimer()

	var totalSearchNanos int64
	var searchedMoves int

	for i := 0; i < b.N; i++ {
		var pos PositionNG
		pos.Set(sequentialSearchBenchmarkFEN)
		TTClear()
		MHTClear()

		b.StartTimer()
		start := time.Now()
		moves := runSequentialSearchBenchmarkMoves(b, &pos, search)
		elapsed := time.Since(start)
		b.StopTimer()

		totalSearchNanos += elapsed.Nanoseconds()
		searchedMoves += moves
	}

	if searchedMoves > 0 {
		b.ReportMetric(float64(totalSearchNanos)/float64(searchedMoves)/1e6, "ms/move")
	}
}

func benchmarkSearchCountersFixedFEN(b *testing.B, depth uint8, search func(*PositionNG) (MoveNG, Value)) {
	b.Helper()
	b.ReportAllocs()
	b.StopTimer()

	var totalPositions uint64
	var totalEvaluates uint64
	var searchedMoves int

	for i := 0; i < b.N; i++ {
		var pos PositionNG
		pos.Set(sequentialSearchBenchmarkFEN)
		TTClear()
		MHTClear()

		visitedPositionCount.Store(0)
		evaluateCallCount.Store(0)

		b.StartTimer()
		moves := runSequentialSearchBenchmarkMoves(b, &pos, search)
		b.StopTimer()

		totalPositions += visitedPositionCount.Load()
		totalEvaluates += evaluateCallCount.Load()
		searchedMoves += moves
	}

	if b.N > 0 {
		b.ReportMetric(float64(totalPositions)/float64(b.N), "positions/sequence")
		b.ReportMetric(float64(totalEvaluates)/float64(b.N), "evals/sequence")
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
	expectedMove, expectedScore := benchmarkAlphaBetaExpected(depth)
	for i := 0; i < b.N; i++ {

		foundMove, foundScore := benchmarkSearchResult(depth, search)

		if foundMove == expectedMove || foundScore == expectedScore {
			success++
		}
	}

	if b.N > 0 {
		b.ReportMetric(float64(success)/float64(b.N)*100, "accuracy_percent")
	}
}

func benchmarkAlphaBetaExpected(depth uint8) (MoveNG, Value) {
	var pos PositionNG
	pos.Set(sequentialSearchBenchmarkFEN)
	TTClear()
	MHTClear()
	return pos.SearchPosition_ab(depth)
}

func benchmarkSearchResult(depth uint8, search func(*PositionNG) (MoveNG, Value)) (MoveNG, Value) {
	var pos PositionNG
	pos.Set(sequentialSearchBenchmarkFEN)
	TTClear()
	MHTClear()
	return search(&pos)
}

func runSequentialSearchBenchmarkMoves(b *testing.B, pos *PositionNG, search func(*PositionNG) (MoveNG, Value)) int {
	b.Helper()

	for i := 0; i < sequentialSearchBenchmarkMoves; i++ {
		move, score := search(pos)
		if !IsOKMove(move) {
			b.Fatalf("search returned invalid move at move %d", i+1)
		}

		var st StateInfo
		pos.DoMove(move, &st)

		sequentialSearchBenchmarkMoveSink = move
		sequentialSearchBenchmarkScoreSink = score
	}

	return sequentialSearchBenchmarkMoves
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
	clearSearch(&pos)

	return &pos
}

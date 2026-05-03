package engine

import (
	"context"
	"testing"
	"time"

	"github.com/hmgle/godogpaw/pikafish"
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
var sequentialSearchBenchmarkLazySMPThreads = []int{4, 5, 6}

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
				benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
					return pos.SearchPosition_ab(depth)
				})
			})
			// b.Run("YBWC", func(b *testing.B) {
			// 	benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
			// 		return pos.SearchPositionYBWC(depth)
			// 	})
			// })
			b.Run("LazySMP", func(b *testing.B) {
				for _, threads := range sequentialSearchBenchmarkLazySMPThreads {
					threads := threads
					b.Run("threads_"+itoaBenchmark(threads), func(b *testing.B) {
						benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
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
			b.Run("Pikafish", func(b *testing.B) {
				pika, err := pikafish.NewFromEnv()
				if err != nil {
					b.Skipf("Pikafish is not available: %v", err)
				}
				defer pika.Close()

				benchmarkSequentialSearchMoves(b, func(pos *PositionNG) (MoveNG, Value) {
					timeout := time.Duration(max(depth, 1)) * time.Minute
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					result, err := pika.BestMove(ctx, pos.FEN(), int(depth))
					if err != nil {
						b.Fatalf("Pikafish search failed: %v", err)
					}
					move, err := ParseUCIMove(pos, result.BestMove)
					if err != nil {
						b.Fatalf("Pikafish returned invalid move %q: %v", result.BestMove, err)
					}
					if !result.HasScore {
						return move, VALUE_DRAW
					}
					return move, Value(result.Score)

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
			// b.Run("YBWC", func(b *testing.B) {
			// 	benchmarkSearchCountersFixedFEN(b, func(pos *PositionNG) (MoveNG, Value) {
			// 		return pos.SearchPositionYBWC(depth)
			// 	})
			// })
			b.Run("LazySMP", func(b *testing.B) {
				for _, threads := range sequentialSearchBenchmarkLazySMPThreads {
					threads := threads
					b.Run("threads_"+itoaBenchmark(threads), func(b *testing.B) {
						benchmarkSearchCountersFixedFEN(b, func(pos *PositionNG) (MoveNG, Value) {
							return pos.SearchPositionLazySMP(depth, threads)
						})
					})
				}
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

	for i := 0; i < b.N; i++ {
		var pos PositionNG
		pos.Set(sequentialSearchBenchmarkFEN)
		TTClear()
		MHTClear()

		elapsed := runSequentialSearchBenchmarkMoves(b, &pos, search)

		totalSearchNanos += elapsed.Nanoseconds()
		searchedMoves += sequentialSearchBenchmarkMoves
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
	var totalElapsed time.Duration

	for i := 0; i < b.N; i++ {
		var pos PositionNG
		pos.Set(sequentialSearchBenchmarkFEN)
		TTClear()
		MHTClear()

		visitedPositionCount.Store(0)
		evaluateCallCount.Store(0)

		elapsed := runSequentialSearchBenchmarkMoves(b, &pos, search)

		totalElapsed += elapsed
		totalPositions += visitedPositionCount.Load()
		totalEvaluates += evaluateCallCount.Load()
		searchedMoves += sequentialSearchBenchmarkMoves
	}

	if b.N > 0 {
		b.ReportMetric(float64(totalPositions)/float64(b.N), "positions/sequence")
		b.ReportMetric(float64(totalEvaluates)/float64(b.N), "evals/sequence")
	}
	if searchedMoves > 0 {
		b.ReportMetric(float64(totalElapsed.Nanoseconds())/float64(searchedMoves)/1e6, "ms/move")
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
		foundMove, foundScore := benchmarkSearchResult(search)

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

func benchmarkSearchResult(search func(*PositionNG) (MoveNG, Value)) (MoveNG, Value) {
	var pos PositionNG
	pos.Set(sequentialSearchBenchmarkFEN)
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

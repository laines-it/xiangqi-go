package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hmgle/godogpaw/engine"
)

type CallResult struct {
	CallNumber  int           `json:"call_number"`
	Depth       int           `json:"depth"`
	Duration    time.Duration `json:"duration_ns"`
	DurationSec float64       `json:"duration_sec"`
}

type FunctionResults struct {
	Name       string       `json:"name"`
	Depth      int          `json:"depth"`
	Results    []CallResult `json:"results"`
	AverageSec float64      `json:"average_sec"`
	TotalSec   float64      `json:"total_sec"`
	Cumulative []float64    `json:"cumulative_sec"`
}

func test(name string, getBestMoveFunc func(i int, pos *engine.PositionNG, depth int) engine.Value, pos *engine.PositionNG, depth int, n int) FunctionResults {
	results := make([]CallResult, 0, n)
	var totalDuration time.Duration
	cumulative := make([]float64, n)

	for i := range n {
		start := time.Now()
		score := getBestMoveFunc(i, pos, depth)
		duration := time.Since(start)

		results = append(results, CallResult{
			CallNumber:  i + 1,
			Depth:       depth,
			Duration:    duration,
			DurationSec: duration.Seconds(),
		})
		totalDuration += duration

		cumulative[i] = totalDuration.Seconds()

		fmt.Printf("Ход %d: %v\n (score: %d)", i+1, duration, score)
	}

	avgDuration := totalDuration / time.Duration(n)
	stats := FunctionResults{
		Name:       name,
		Depth:      depth,
		Results:    results,
		AverageSec: avgDuration.Seconds(),
		TotalSec:   totalDuration.Seconds(),
		Cumulative: cumulative,
	}

	return stats
}

func main() {
	test := false
	if test {
		test_speed()
		return
	}
	var pos engine.PositionNG
	//initialFen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	fen1 := "1C2ka3/9/C1Nab1n2/p3p3p/6p2/9/P3P3P/3AB4/3p2c2/c1BAK4 w - - 0 1"
	fen2 := "rnbakabCr/9/7c1/p1p1p1p2/1c6p/9/P1P1P1P1P/9/1C2K4/RNBA1ABNR b - - 0 1"
	fen3 := "rnbaka1nr/9/3c4b/p1p1p1p1p/7C1/9/P1P1P1P1P/1c7/9/RNBAKABNR w - - 0 1"
	fen4 := "rCbakab1r/9/6nc1/2p1p3p/p5p2/1c7/P1P1P1PCP/R8/9/1NBAKABNR b - - 0 1"

	fen5 := "1nbakCbr1/r8/1c4c2/p1p3p1p/4p4/8P/P1P1P1P2/1CN6/8R/R1BAKABN1 w - - 0 1"

	fen6 := "r1bk1abnr/5c3/9/p1p1pC2p/3C5/9/P1P1P1P2/8B/9/RNBAKA1Nc b - - 0 1"

	fen7 := "2b2kb2/9/r7r/pc2p1p1p/2p6/9/P1P1c1P1P/9/9/RNBAKABNR w - - 0 1"

	fen8 := "1nbakabr1/9/r8/p1pC2p1p/9/9/P1P1p1P1P/2N1B1Nc1/4A4/1R1cK1B1R b - - 0 1"

	fen9 := "rnbakabn1/8C/1r7/p1p3p1p/4p4/2P6/P3P1P1P/7C1/2R6/2B1K1c1R w - - 0 1"

	fen10 := "1rbaka3/5R3/4b3r/p1p3p1p/4P4/4c4/P1P3P1P/4C4/9/RNBK2B2 b - - 0 1"

	fen11 := "2b1kabnr/R3aC3/2n6/2p1p1p2/9/4P3p/2P3P2/9/2c1A4/1NB2KB1R w - - 0 1"

	pos.Set(fen1)
	fmt.Println(pos.String(true))
	pos.Set(fen2)
	fmt.Println(pos.String(true))
	pos.Set(fen3)
	fmt.Println(pos.String(true))
	pos.Set(fen4)
	fmt.Println(pos.String(true))
	pos.Set(fen5)
	fmt.Println(pos.String(true))
	pos.Set(fen6)
	fmt.Println(pos.String(true))
	pos.Set(fen7)
	fmt.Println(pos.String(true))
	pos.Set(fen8)
	fmt.Println(pos.String(true))
	pos.Set(fen9)
	fmt.Println(pos.String(true))
	pos.Set(fen10)
	fmt.Println(pos.String(true))
	pos.Set(fen11)
	fmt.Println(pos.String(true))
	// // Magic example
	// emptyFen := "4k4/7H1/9/1H7/9/9/1r2H4/9/9/3K5 w - - 0 1"
	// pos.Set(emptyFen)
	// fmt.Println(pos.String(light))

	// Match example
	//startMatch("YBWC", GetBestMoveYBWC, 6, "LazySMP", GetBestMoveLazy5, 6)

	// Accuracy example
	// name := "YBWC"
	// accuracy := checkAccuracy(name, 2)
	// fmt.Printf("Точность %s, depth 2: %.2f\n", name, accuracy)
	// accuracy = checkAccuracy(name, 7)
	// fmt.Printf("Точность %s, depth 7: %.2f\n", name, accuracy)
	// accuracy = checkAccuracy(name, 8)
	// fmt.Printf("Точность %s, depth 8: %.2f\n", name, accuracy)
}

func generateRandomPosition() engine.PositionNG {
	var pos engine.PositionNG
	pos.Set("rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1")
	moves := make([]engine.MoveNG, engine.MAX_MOVES)
	for range 10 {
		size := pos.GenerateLEGAL(moves)
		if size == 0 {
			break
		}
		randomMove := moves[rand.Intn(len(moves[:size]))]
		var st engine.StateInfo
		pos.DoMove(randomMove, &st)
	}
	return pos
}

func checkAccuracy(name string, depth int) float64 {
	n := 20
	success := 0
	for range n {
		pos := generateRandomPosition()

		expectedMove, expectedScore := pos.SearchPosition_ab(uint8(depth))
		var foundMove engine.MoveNG
		var foundScore engine.Value
		switch name {
		case "Lazy":
			foundMove, foundScore = pos.SearchPositionLazySMP(uint8(depth), 0)
		case "YBWC":
			foundMove, foundScore = pos.SearchPositionYBWC(uint8(depth))
		case "Alpha-Beta":
			foundMove, foundScore = pos.SearchPosition_ab(uint8(depth))
		default:
			log.Fatalf("Unknown search method: %s", name)
			return 0
		}
		if foundMove == expectedMove {
			success++
			fmt.Println("found success")
		} else {
			if foundScore != expectedScore {
				fmt.Printf("found failure, expected score: %d, found score: %d\n", expectedScore, foundScore)
			} else {
				success++
				fmt.Printf("found inaccuracy, expected: %s, found: %s\n", pos.MoveStr(expectedMove), pos.MoveStr(foundMove))
			}
		}
	}
	return float64(success) / float64(n)
}

func startMatch(name1 string, engine1 func(i int, pos *engine.PositionNG, depth int) engine.Value, depth1 int,
	name2 string, engine2 func(i int, pos *engine.PositionNG, depth int) engine.Value, depth2 int) {
	initialFen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	var pos engine.PositionNG
	pos.Set(initialFen)

	value := engine.VALUE_DRAW
	i := 0

	result := 0
	gameNumber := 1

	for gameNumber < 5 {
		for !isFinished(value) {
			if i > int(engine.MAX_MOVES) {
				fmt.Println("Превышено максимальное количество ходов.")
				break
			}
			if i%2 == 0 {
				value = engine1(i, &pos, depth1)
				fmt.Printf("%s Score: %d\n", name1, value)
			} else {
				value = engine2(i, &pos, depth2)
				fmt.Printf("%s Score: %d\n", name2, value)
			}
			fmt.Println(pos.String(true))
			i++
		}
		if i%2 == 0 {
			result += 1
			fmt.Printf("Game %d: Победитель: %s\n", gameNumber, name2)
		} else {
			result -= 1
			fmt.Printf("Game %d: Победитель: %s\n", gameNumber, name1)
		}
		gameNumber++
		pos.Set(initialFen)
		value = engine.VALUE_DRAW
		i = 0
	}
	if result > 0 {
		fmt.Printf("Матч завершён. Победитель: %s\n", name2)
	} else if result < 0 {
		fmt.Printf("Матч завершён. Победитель: %s\n", name1)
	} else {
		fmt.Printf("Матч завершён. Ничья\n")
	}
}

func isFinished(value engine.Value) bool {
	return value <= -engine.VALUE_MATE+1 ||
		value >= engine.VALUE_MATE-1
}

func test_speed() {
	pos := generateRandomPosition()
	fmt.Println("Исходная позиция:")
	fmt.Println(pos.String(true))

	n := 4

	newPos := pos
	depth := 4
	fmt.Println("Alpha-beta, Тестирование на глубине", depth)
	par4 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)
	newPos = pos
	fmt.Println("Lazy, Тестирование на глубине", depth)
	ab4 := test("Lazy 5", GetBestMoveLazy5, &newPos, depth, n)

	newPos = pos
	depth = 6
	fmt.Println("Alpha-beta, Тестирование на глубине", depth)
	par6 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)
	newPos = pos
	fmt.Println("Lazy, Тестирование на глубине", depth)
	ab6 := test("Lazy 5", GetBestMoveLazy5, &newPos, depth, n)

	newPos = pos
	depth = 8
	fmt.Println("Alpha-beta, Тестирование на глубине", depth)
	par8 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)
	newPos = pos
	fmt.Println("Lazy, Тестирование на глубине", depth)
	ab8 := test("Lazy 5", GetBestMoveLazy5, &newPos, depth, n)

	newPos = pos
	depth = 10
	fmt.Println("Alpha-beta, Тестирование на глубине", depth)
	par10 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)
	newPos = pos
	fmt.Println("Lazy, Тестирование на глубине", depth)
	ab10 := test("Lazy 5", GetBestMoveLazy5, &newPos, depth, n)

	allData := struct {
		Functions []FunctionResults `json:"functions"`
		N         int               `json:"n"`
		Depth     int               `json:"depth"`
	}{
		Functions: []FunctionResults{par4, ab4, par6, ab6, par8, ab8, par10, ab10},
		N:         n,
		Depth:     depth,
	}

	jsonData, err := json.MarshalIndent(allData, "", "  ")
	if err != nil {
		log.Fatal("Ошибка JSON:", err)
	}
	err = os.WriteFile("three_searches_results.json", jsonData, 0644)
	if err != nil {
		log.Fatal("Ошибка записи файла:", err)
	}

	fmt.Println("\nДанные сохранены в three_searches_results.json")
}

func GetBestMove(i int, pos *engine.PositionNG, depth int) {
	var state engine.StateInfo
	var mv engine.MoveNG
	d := uint8(depth)
	mv = pos.SearchPosition(d)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
}

func GetBestMoveLazyMAXPROCS(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo
	d := uint8(depth)
	mv, score := pos.SearchPositionLazySMP(d, 0)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	return score
}

func GetBestMoveLazy4(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo
	d := uint8(depth)
	mv, score := pos.SearchPositionLazySMP(d, 4)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	return score
}

func GetBestMoveLazy5(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo
	d := uint8(depth)
	mv, score := pos.SearchPositionLazySMP(d, 5)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	return score
}

func GetBestMoveAb(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo
	d := uint8(depth)
	mv, score := pos.SearchPosition_ab(d)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	return score
}

func GetBestMovePar(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo
	d := uint8(depth)
	mv, score := pos.ParallelSearch(d)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	fmt.Println(pos.String(true))
	return score
}

func GetBestMoveYBWC(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo
	d := uint8(depth)
	mv, score := pos.SearchPositionYBWC(d)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	return score
}

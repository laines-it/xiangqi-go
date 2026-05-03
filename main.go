package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/hmgle/godogpaw/engine"
	"github.com/hmgle/godogpaw/pikafish"
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

var defaultPikafish struct {
	once   sync.Once
	client *pikafish.Engine
	err    error
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

var pika *pikafish.Engine

func main() {
	test := false
	if test {
		test_speed()
		return
	}

	// // Magic example
	// var pos engine.PositionNG
	// emptyFen := "4k4/7H1/9/1H7/9/9/1r2H4/9/9/3K5 w - - 0 1"
	// pos.Set(emptyFen)
	// fmt.Println(pos.String(true))
	// sq := engine.SQ_B3
	// occupied := engine.SquareBB[engine.SQ_B6].
	// 	Or(engine.SquareBB[engine.SQ_E3]).
	// 	Or(engine.SquareBB[engine.SQ_H8])
	// info := engine.MagicInfo(engine.ROOK, sq, occupied)
	// fmt.Printf("mask_popcount=%d shift=%d table_variants=%d\n", info.MaskPopcount, info.Shift, info.TableVariants)
	// fmt.Printf("magic_hi=0x%016X magic_lo=0x%016X\n", info.Magic.Hi, info.Magic.Lo)
	// fmt.Printf("index=%d\n", info.Index)

	//Match example
	var err error
	pika, err = getDefaultPikafish()
	if err != nil {
		log.Fatalf("Failed to initialize Pikafish: %v", err)
	}

	startMatch("YBWC", GetBestMoveYBWC, 6, "Pikafish", GetBestMovePikafish, 1)

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

func GetBestMovePikafish(i int, pos *engine.PositionNG, depth int) engine.Value {
	var state engine.StateInfo

	client := pika
	if client == nil {
		var err error
		client, err = getDefaultPikafish()
		if err != nil {
			log.Fatalf("Pikafish initialization failed: %v", err)
		}
	}

	timeout := time.Duration(max(depth, 1)) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := client.BestMove(ctx, pos.FEN(), depth)
	if err != nil {
		log.Fatalf("Pikafish search failed: %v", err)
	}
	mv, err := engine.ParseUCIMove(pos, result.BestMove)
	if err != nil {
		log.Fatalf("Pikafish returned invalid move %q: %v", result.BestMove, err)
	}
	score := engine.VALUE_DRAW
	if result.HasScore {
		score = engine.Value(result.Score)
	}

	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
	return score
}

func getDefaultPikafish() (*pikafish.Engine, error) {
	defaultPikafish.once.Do(func() {
		defaultPikafish.client, defaultPikafish.err = pikafish.NewFromEnv()
	})
	return defaultPikafish.client, defaultPikafish.err
}

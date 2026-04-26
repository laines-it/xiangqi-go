package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/hmgle/godogpaw/engine"
	"github.com/sirupsen/logrus"
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
		getBestMoveFunc(i, pos, depth)
		duration := time.Since(start)

		results = append(results, CallResult{
			CallNumber:  i + 1,
			Depth:       depth,
			Duration:    duration,
			DurationSec: duration.Seconds(),
		})
		totalDuration += duration

		cumulative[i] = totalDuration.Seconds()

		fmt.Printf("Ход %d: %v\n", i+1, duration)
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
	initialFen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	var pos engine.PositionNG
	pos.Set(initialFen)

	value := engine.VALUE_DRAW
	// Parallel vs Alpha-Beta
	i := 0
	for -engine.VALUE_MATE < value && value < engine.VALUE_MATE {
		scorePar := GetBestMovePar(i, &pos, 4)
		fmt.Printf("Parallel Search Score: %d\n", scorePar)
		fmt.Println(pos.String())
		scoreAb := GetBestMoveAb(i, &pos, 4)
		fmt.Printf("Alpha-Beta Search Score: %d\n", scoreAb)
		fmt.Println(pos.String())
		value = pos.Evaluate()
		i++
	}
	fmt.Println(pos.String())
	if value > 0 {
		fmt.Println("Победа Parallel!")
	} else if value < 0 {
		fmt.Println("Победа Alpha-Beta!")
	} else {
		fmt.Println("Ничья!")
	}
}

func test_speed() {
	initialFen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	var pos engine.PositionNG
	pos.Set(initialFen)

	for i := range 10 {
		GetBestMove(i, &pos, 2)
	}

	n := 10

	newPos := pos
	depth := 4
	par4 := test("Parallel", GetBestMovePar, &newPos, depth, n)
	newPos = pos
	ab4 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)

	newPos = pos
	depth = 6
	par6 := test("Parallel", GetBestMovePar, &newPos, depth, n)
	newPos = pos
	ab6 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)

	newPos = pos
	depth = 8
	par8 := test("Parallel", GetBestMovePar, &newPos, depth, n)
	newPos = pos
	ab8 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)

	newPos = pos
	depth = 10
	par10 := test("Parallel", GetBestMovePar, &newPos, depth, n)
	newPos = pos
	ab10 := test("Alpha-Beta", GetBestMoveAb, &newPos, depth, n)

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
	return score
}

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	logPath := "/tmp/godogpaw-ucci.log"
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}

	mw := io.MultiWriter(os.Stderr, f)
	logrus.SetOutput(mw)
	log.SetOutput(mw)
	logrus.SetLevel(logrus.DebugLevel)
}

func logPanic() {
	if r := recover(); r != nil {
		logrus.WithFields(logrus.Fields{
			"panic": r,
			"stack": string(debug.Stack()),
		}).Error("engine panic")
		panic(r)
	}
}

//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"syscall/js"
	"time"

	"github.com/hmgle/godogpaw/engine"
)

const startFEN = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

const (
	algorithmAlphaBeta = "alpha-beta"
	algorithmYBWC      = "ybwc"
	algorithmLazySMP   = "lazysmp"
	algorithmPikafish  = "pikafish"
)

var pos engine.PositionNG
var moveHistory []moveRecord

type moveRecord struct {
	move engine.MoveNG
	st   engine.StateInfo
}

type boardState struct {
	Board      [engine.SQUARE_NB]int `json:"board"`
	SideToMove int                   `json:"sideToMove"`
	InCheck    bool                  `json:"inCheck"`
	IsGameOver bool                  `json:"isGameOver"`
	LastFrom   int                   `json:"lastMoveFrom"`
	LastTo     int                   `json:"lastMoveTo"`
	LastMove   string                `json:"lastMove"`
	FEN        string                `json:"fen"`
	MoveCount  int                   `json:"moveCount"`
}

type moveAction struct {
	OK      bool   `json:"ok"`
	Move    string `json:"move,omitempty"`
	From    int    `json:"from,omitempty"`
	To      int    `json:"to,omitempty"`
	Message string `json:"message,omitempty"`
}

type searchResult struct {
	OK                 bool     `json:"ok"`
	Move               string   `json:"move,omitempty"`
	From               int      `json:"from,omitempty"`
	To                 int      `json:"to,omitempty"`
	Score              int32    `json:"score"`
	Depth              int      `json:"depth"`
	Threads            int      `json:"threads,omitempty"`
	Algorithm          string   `json:"algorithm"`
	RequestedAlgorithm string   `json:"requestedAlgorithm"`
	Fallback           bool     `json:"fallback"`
	Message            string   `json:"message,omitempty"`
	DurationMs         float64  `json:"durationMs"`
	PV                 []string `json:"pv,omitempty"`
}

type algorithmInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Fallback  string `json:"fallback,omitempty"`
	Note      string `json:"note,omitempty"`
}

type engineInfo struct {
	Algorithms []algorithmInfo `json:"algorithms"`
	GOOS       string          `json:"goos"`
	GOARCH     string          `json:"goarch"`
	MaxProcs   int             `json:"maxProcs"`
}

func engineNewGame(_ js.Value, args []js.Value) any {
	fen := startFEN
	if len(args) > 0 {
		value := strings.TrimSpace(args[0].String())
		if value != "" {
			fen = value
		}
	}

	pos.Set(fen)
	moveHistory = nil
	return mustJSON(moveAction{OK: true})
}

func engineGetBoard(_ js.Value, _ []js.Value) any {
	var st boardState
	for i := 0; i < engine.SQUARE_NB; i++ {
		st.Board[i] = pos.Board[i]
	}
	st.SideToMove = int(pos.SideToMove)
	st.InCheck = pos.Checkers().IsNotZero()
	st.IsGameOver = legalMoveCount() == 0
	st.LastFrom = -1
	st.LastTo = -1
	if len(moveHistory) > 0 {
		last := moveHistory[len(moveHistory)-1].move
		if engine.IsOKMove(last) {
			st.LastFrom = engine.FromSQ(last)
			st.LastTo = engine.ToSQ(last)
			st.LastMove = engine.Move2Str(last)
		}
	}
	st.FEN = pos.FEN()
	st.MoveCount = len(moveHistory)
	return mustJSON(st)
}

func engineGetLegalMovesFrom(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return "[]"
	}
	sq := args[0].Int()
	if sq < 0 || sq >= engine.SQUARE_NB {
		return "[]"
	}

	var list [engine.MAX_MOVES]engine.MoveNG
	size := pos.GenerateLEGAL(list[:])
	targets := make([]int, 0, 8)
	for i := uint8(0); i < size; i++ {
		if engine.FromSQ(list[i]) == sq {
			targets = append(targets, engine.ToSQ(list[i]))
		}
	}
	return mustJSON(targets)
}

func engineDoMoveBySquares(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return mustJSON(moveAction{OK: false, Message: "Missing move squares."})
	}

	from := args[0].Int()
	to := args[1].Int()
	if from < 0 || from >= engine.SQUARE_NB || to < 0 || to >= engine.SQUARE_NB {
		return mustJSON(moveAction{OK: false, Message: "Square is out of range."})
	}

	m := engine.MakeMove(from, to)
	if !isLegalMove(m) {
		return mustJSON(moveAction{OK: false, Message: "Illegal move."})
	}

	doMove(m)
	return mustJSON(moveAction{
		OK:   true,
		Move: engine.Move2Str(m),
		From: from,
		To:   to,
	})
}

func engineUndoMove(_ js.Value, _ []js.Value) any {
	if len(moveHistory) == 0 {
		return false
	}
	last := moveHistory[len(moveHistory)-1]
	pos.UndoMove(last.move)
	moveHistory = moveHistory[:len(moveHistory)-1]
	return true
}

func engineSearch(_ js.Value, args []js.Value) any {
	algorithm, depth, threads := parseSearchArgs(args)

	handler := js.FuncOf(func(_ js.Value, promiseArgs []js.Value) any {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			result, err := searchAndMove(algorithm, depth, threads)
			if err != nil {
				reject.Invoke(err.Error())
				return
			}
			resolve.Invoke(mustJSON(result))
		}()
		return nil
	})

	return js.Global().Get("Promise").New(handler)
}

func engineGetEngineInfo(_ js.Value, _ []js.Value) any {
	return mustJSON(engineInfo{
		Algorithms: []algorithmInfo{
			{ID: algorithmAlphaBeta, Name: "Alpha-Beta", Available: true},
			{ID: algorithmYBWC, Name: "YBWC", Available: true},
			{ID: algorithmLazySMP, Name: "LazySMP", Available: true},
			{
				ID:        algorithmPikafish,
				Name:      "Pikafish",
				Available: false,
				Fallback:  algorithmYBWC,
				Note:      "Pikafish is a native UCI process; GitHub Pages falls back to YBWC.",
			},
		},
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		MaxProcs: runtime.GOMAXPROCS(0),
	})
}

func parseSearchArgs(args []js.Value) (string, uint8, int) {
	algorithm := algorithmAlphaBeta
	depth := 4
	threads := runtime.GOMAXPROCS(0)

	if len(args) > 0 {
		if args[0].Type() == js.TypeNumber {
			depth = args[0].Int()
		} else if args[0].Type() == js.TypeString {
			algorithm = normalizeAlgorithm(args[0].String())
		}
	}
	if len(args) > 1 {
		if args[0].Type() == js.TypeNumber {
			// Backward compatibility with the older engineSearch(depth, timeMs).
		} else if args[1].Type() == js.TypeNumber {
			depth = args[1].Int()
		}
	}
	if len(args) > 2 && args[2].Type() == js.TypeNumber {
		threads = args[2].Int()
	}

	if depth < 1 {
		depth = 1
	}
	if depth > 12 {
		depth = 12
	}
	if threads < 1 {
		threads = 1
	}
	if threads > 16 {
		threads = 16
	}

	return algorithm, uint8(depth), threads
}

func normalizeAlgorithm(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alpha", "ab", "alpha-beta", "alphabeta":
		return algorithmAlphaBeta
	case "ybwc":
		return algorithmYBWC
	case "lazy", "lazy-smp", "lazysmp":
		return algorithmLazySMP
	case "pika", "pikafish":
		return algorithmPikafish
	default:
		return algorithmAlphaBeta
	}
}

func searchAndMove(requested string, depth uint8, threads int) (result searchResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("search failed: %v", r)
		}
	}()

	result = searchResult{
		Depth:              int(depth),
		Threads:            threads,
		Algorithm:          requested,
		RequestedAlgorithm: requested,
	}

	if legalMoveCount() == 0 {
		result.Message = "No legal moves."
		return result, nil
	}

	start := time.Now()
	var bestMove engine.MoveNG
	var score engine.Value

	switch requested {
	case algorithmAlphaBeta:
		bestMove, score = pos.SearchPosition_ab(depth)
	case algorithmYBWC:
		bestMove, score = pos.SearchPositionYBWC(depth)
	case algorithmLazySMP:
		bestMove, score = pos.SearchPositionLazySMP(depth, threads)
	case algorithmPikafish:
		bestMove, score = pos.SearchPositionYBWC(depth)
		result.Algorithm = algorithmYBWC
		result.Fallback = true
		result.Message = "Pikafish needs a native UCI process; this browser build used YBWC."
	default:
		bestMove, score = pos.SearchPosition_ab(depth)
		result.Algorithm = algorithmAlphaBeta
		result.Fallback = true
		result.Message = "Unknown algorithm; this move used Alpha-Beta."
	}

	result.DurationMs = float64(time.Since(start).Microseconds()) / 1000
	result.Score = int32(score)
	result.PV = pvStrings()

	if !engine.IsOKMove(bestMove) || !pos.Legal(bestMove) {
		bestMove = firstLegalMove()
		if !engine.IsOKMove(bestMove) {
			result.Message = "No legal moves."
			return result, nil
		}
		if result.Message == "" {
			result.Message = "Search did not return a legal move; played the first legal move."
		}
		result.Fallback = true
	}

	result.OK = true
	result.Move = engine.Move2Str(bestMove)
	result.From = engine.FromSQ(bestMove)
	result.To = engine.ToSQ(bestMove)
	doMove(bestMove)
	return result, nil
}

func pvStrings() []string {
	length := engine.PvLength[0]
	if length <= 0 {
		return nil
	}

	pv := make([]string, 0, length)
	for i := 0; i < length && i < len(engine.PvTable); i++ {
		move := engine.PvTable[i]
		if !engine.IsOKMove(move) {
			break
		}
		pv = append(pv, engine.Move2Str(move))
	}
	return pv
}

func doMove(move engine.MoveNG) {
	var rec moveRecord
	rec.move = move
	pos.DoMove(move, &rec.st)
	moveHistory = append(moveHistory, rec)
}

func isLegalMove(move engine.MoveNG) bool {
	var list [engine.MAX_MOVES]engine.MoveNG
	size := pos.GenerateLEGAL(list[:])
	for i := uint8(0); i < size; i++ {
		if list[i] == move {
			return true
		}
	}
	return false
}

func firstLegalMove() engine.MoveNG {
	var list [engine.MAX_MOVES]engine.MoveNG
	size := pos.GenerateLEGAL(list[:])
	if size == 0 {
		return engine.MOVE_NONE
	}
	return list[0]
}

func legalMoveCount() int {
	var list [engine.MAX_MOVES]engine.MoveNG
	return int(pos.GenerateLEGAL(list[:]))
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func main() {
	g := js.Global()
	g.Set("engineNewGame", js.FuncOf(engineNewGame))
	g.Set("engineGetBoard", js.FuncOf(engineGetBoard))
	g.Set("engineGetLegalMovesFrom", js.FuncOf(engineGetLegalMovesFrom))
	g.Set("engineDoMoveBySquares", js.FuncOf(engineDoMoveBySquares))
	g.Set("engineUndoMove", js.FuncOf(engineUndoMove))
	g.Set("engineSearch", js.FuncOf(engineSearch))
	g.Set("engineGetEngineInfo", js.FuncOf(engineGetEngineInfo))

	pos.Set(startFEN)
	select {}
}

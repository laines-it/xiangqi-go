package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"

	"github.com/hmgle/godogpaw/engine"
	"github.com/sirupsen/logrus"
)

func main() {
	initialFen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	var pos engine.PositionNG
	pos.Set(initialFen)
	for i := range 10 {
		GetBestMove(i, &pos)
		fmt.Println(pos.String())
	}
}

func GetBestMove(i int, pos *engine.PositionNG) {
	var state engine.StateInfo
	var mv engine.MoveNG
	mv = pos.SearchPosition(8)
	fmt.Println(i, pos.MoveStr(mv))
	pos.DoMove(mv, &state)
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

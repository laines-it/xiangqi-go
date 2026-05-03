package pikafish

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPath           = "pikafish"
	defaultStartupTimeout = 5 * time.Second
	valueDraw             = int32(0)
	valueMate             = int32(32000)
)

type Config struct {
	Path           string
	Options        map[string]string
	StartupTimeout time.Duration
}

type SearchResult struct {
	BestMove string
	Score    int32
	HasScore bool
}

type Engine struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	mu      sync.Mutex
	closed  bool
	killErr error
}

func NewFromEnv() (*Engine, error) {
	path := os.Getenv("PIKAFISH_PATH")
	if path == "" {
		path = resolveDefaultPath()
	}
	return New(Config{Path: path})
}

func New(config Config) (*Engine, error) {
	if config.Path == "" {
		config.Path = defaultPath
	}
	if config.StartupTimeout <= 0 {
		config.StartupTimeout = defaultStartupTimeout
	}

	cmdPath := config.Path
	cmdDir := ""
	if strings.ContainsAny(config.Path, `/\`) {
		absPath, err := filepath.Abs(config.Path)
		if err != nil {
			return nil, err
		}
		cmdPath = absPath
		cmdDir = workingDirForExecutable(absPath)
	}

	cmd := exec.Command(cmdPath)
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start pikafish: %w", err)
	}

	e := &Engine{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
	}
	e.stdout.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	ctx, cancel := context.WithTimeout(context.Background(), config.StartupTimeout)
	defer cancel()

	if err := e.sendLine("uci"); err != nil {
		_ = e.Close()
		return nil, err
	}
	if _, err := e.readUntil(ctx, func(line string) (bool, error) {
		return line == "uciok", nil
	}); err != nil {
		_ = e.Close()
		return nil, fmt.Errorf("initialize pikafish UCI: %w", err)
	}

	for name, value := range config.Options {
		if err := e.sendLine("setoption name %s value %s", name, value); err != nil {
			_ = e.Close()
			return nil, err
		}
	}

	if err := e.sendLine("isready"); err != nil {
		_ = e.Close()
		return nil, err
	}
	if _, err := e.readUntil(ctx, func(line string) (bool, error) {
		return line == "readyok", nil
	}); err != nil {
		_ = e.Close()
		return nil, fmt.Errorf("wait for pikafish readiness: %w", err)
	}

	return e, nil
}

func (e *Engine) BestMove(ctx context.Context, fen string, depth int) (SearchResult, error) {
	if depth <= 0 {
		depth = 1
	}
	if ctx == nil {
		ctx = context.Background()
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return SearchResult{}, errors.New("pikafish engine is closed")
	}
	if err := e.sendLine("position fen %s", fen); err != nil {
		return SearchResult{}, err
	}
	if err := e.sendLine("go depth %d", depth); err != nil {
		return SearchResult{}, err
	}

	result := SearchResult{Score: valueDraw}
	_, err := e.readUntil(ctx, func(line string) (bool, error) {
		if strings.HasPrefix(line, "info ") {
			if score, ok := parseScore(line); ok {
				result.Score = score
				result.HasScore = true
			}
			return false, nil
		}
		if !strings.HasPrefix(line, "bestmove ") {
			return false, nil
		}

		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] == "(none)" || fields[1] == "none" {
			return false, errors.New("pikafish returned no move")
		}
		result.BestMove = fields[1]
		return true, nil
	})
	if err != nil {
		return SearchResult{}, err
	}
	return result, nil
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return e.killErr
	}
	e.closed = true
	_ = e.sendLine("quit")
	_ = e.stdin.Close()
	err := e.cmd.Wait()
	if err != nil {
		e.killErr = err
	}
	return err
}

func (e *Engine) sendLine(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(e.stdin, format+"\n", args...)
	return err
}

func (e *Engine) readUntil(ctx context.Context, accept func(string) (bool, error)) (string, error) {
	for {
		line, err := e.readLine(ctx)
		if err != nil {
			return "", err
		}
		ok, err := accept(line)
		if err != nil {
			return "", err
		}
		if ok {
			return line, nil
		}
	}
}

func (e *Engine) readLine(ctx context.Context) (string, error) {
	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		if e.stdout.Scan() {
			ch <- readResult{line: strings.TrimSpace(e.stdout.Text())}
			return
		}
		err := e.stdout.Err()
		if err == nil {
			err = io.EOF
		}
		ch <- readResult{err: err}
	}()

	select {
	case result := <-ch:
		return result.line, result.err
	case <-ctx.Done():
		e.killErr = e.cmd.Process.Kill()
		e.closed = true
		return "", ctx.Err()
	}
}

func parseScore(line string) (int32, bool) {
	fields := strings.Fields(line)
	for i := 0; i+2 < len(fields); i++ {
		if fields[i] != "score" {
			continue
		}
		value, err := strconv.Atoi(fields[i+2])
		if err != nil {
			return valueDraw, false
		}
		switch fields[i+1] {
		case "cp":
			return int32(value), true
		case "mate":
			if value > 0 {
				return valueMate - int32(value), true
			}
			return -valueMate - int32(value), true
		}
	}
	return valueDraw, false
}

func resolveDefaultPath() string {
	candidates := findLocalPikafishCandidates()
	if len(candidates) > 0 {
		sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
		return candidates[0]
	}
	return defaultPath
}

func findLocalPikafishCandidates() []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}

	var candidates []string
	for dir := wd; ; dir = filepath.Dir(dir) {
		pattern := filepath.Join(dir, "tools", "pikafish", "Pikafish.*", "Windows", "pikafish-sse41-popcnt.exe")
		matches, err := filepath.Glob(pattern)
		if err == nil {
			candidates = append(candidates, matches...)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return candidates
}

func workingDirForExecutable(path string) string {
	dir := filepath.Dir(path)
	if fileExists(filepath.Join(dir, "pikafish.nnue")) {
		return dir
	}

	parent := filepath.Dir(dir)
	if fileExists(filepath.Join(parent, "pikafish.nnue")) {
		return parent
	}
	return dir
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

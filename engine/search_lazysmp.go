package engine

import (
	"runtime"
	"sync/atomic"
)

const lazySMPTBWinBound = VALUE_MATE_IN_MAX_PLY

type LazySMPOptions struct {
	Threads    int
	MultiPV    int
	MinQSDepth uint8
}

type LazySMPRootMove struct {
	Move     MoveNG
	Score    Value
	PV       [MAX_MOVES]MoveNG
	PVLength int
	Depth    uint8
	Searched bool
}

type lazySMPThreadData struct {
	idx            int
	pos            *PositionNG
	ctx            *SearchContext
	rootMoves      []LazySMPRootMove
	pv             searchPV
	previousScore  Value
	completedDepth uint8
	minQSDepth     uint8
	searchedRoots  int
	aborted        bool
}

type lazySMPWorker struct {
	thread *lazySMPThreadData
	start  chan lazySMPWork
	done   chan struct{}
	stop   chan struct{}
	abort  atomic.Bool
}

type lazySMPWork struct {
	depth uint8
	alpha Value
	beta  Value
}

func (pos *PositionNG) SearchPositionLazySMP(depth uint8, threadCount int) (bestMove MoveNG, score Value) {
	return pos.SearchPositionLazySMPWithOptions(depth, LazySMPOptions{Threads: threadCount})
}

func (pos *PositionNG) SearchPositionLazySMPWithOptions(depth uint8, opts LazySMPOptions) (bestMove MoveNG, score Value) {
	opts = opts.normalized()
	ctx := newSearchContext()
	clearSearch(ctx, pos)

	rootMoves := lazySMPGenerateRootMoves(pos, ctx)
	if depth == 0 || len(rootMoves) == 0 {
		return MOVE_NONE, VALUE_DRAW
	}

	return pos.searchPositionLazySMPSynchronized(ctx, depth, opts, rootMoves)
}

func (pos *PositionNG) searchPositionLazySMPSynchronized(ctx *SearchContext, depth uint8, opts LazySMPOptions, rootMoves []LazySMPRootMove) (bestMove MoveNG, score Value) {
	threads := lazySMPCreateThreads(pos, ctx, opts, rootMoves)
	workers := lazySMPStartWorkers(threads[1:])
	defer lazySMPStopWorkers(workers)

	mainThread := threads[0]

	mainThread.prepareRootSearch(pos, rootMoves)
	mainThread.searchRoot(1, -VALUE_INFINITE, VALUE_INFINITE)
	lazySMPPublishMainPV(mainThread)

	prevScore := mainThread.rootMoves[0].Score
	bestThread := mainThread

	for currentDepth := uint8(2); currentDepth <= depth; currentDepth++ {
		alpha := -VALUE_INFINITE
		beta := VALUE_INFINITE
		if currentDepth > 2 {
			window := Value(40 + 15*int(currentDepth))
			alpha = max(prevScore-window, -VALUE_INFINITE)
			beta = min(prevScore+window, VALUE_INFINITE)
		}

		for {
			lazySMPRunAspirationIteration(pos, threads, workers, mainThread.rootMoves, currentDepth, alpha, beta)
			bestThread = lazySMPVoteBestThread(threads)
			if bestThread == nil || !bestThread.hasBestRootMove() {
				bestThread = threads[0]
			}

			score = bestThread.rootMoves[0].Score
			if currentDepth > 2 && (score <= alpha || score >= beta) &&
				(alpha != -VALUE_INFINITE || beta != VALUE_INFINITE) {
				alpha = -VALUE_INFINITE
				beta = VALUE_INFINITE
				continue
			}

			break
		}

		mainThread = bestThread

		prevScore = mainThread.rootMoves[0].Score
		mainThread.previousScore = prevScore
		lazySMPPublishMainPV(mainThread)
	}

	bestMove = mainThread.rootMoves[0].Move
	score = mainThread.rootMoves[0].Score
	return bestMove, score
}

func (opts LazySMPOptions) normalized() LazySMPOptions {
	if opts.Threads <= 0 {
		opts.Threads = runtime.GOMAXPROCS(0)
	}
	if opts.Threads < 1 {
		opts.Threads = 1
	}
	if opts.MultiPV < 1 {
		opts.MultiPV = 1
	}
	return opts
}

func lazySMPCreateThreads(pos *PositionNG, rootCtx *SearchContext, opts LazySMPOptions, rootMoves []LazySMPRootMove) []*lazySMPThreadData {
	threads := make([]*lazySMPThreadData, opts.Threads)
	for idx := range threads {
		threadPos := pos
		threadCtx := rootCtx
		if idx > 0 {
			threadCtx = newSearchContext()
			threadPos = lazySMPCopyPosition(pos, threadCtx)
		}

		threads[idx] = &lazySMPThreadData{
			idx:        idx,
			pos:        threadPos,
			ctx:        threadCtx,
			minQSDepth: opts.MinQSDepth,
		}
		threads[idx].setRootMoves(rootMoves)
	}
	return threads
}

func lazySMPStartWorkers(threads []*lazySMPThreadData) []*lazySMPWorker {
	workers := make([]*lazySMPWorker, len(threads))
	for i, thread := range threads {
		worker := &lazySMPWorker{
			thread: thread,
			start:  make(chan lazySMPWork),
			done:   make(chan struct{}, 1),
			stop:   make(chan struct{}),
		}
		workers[i] = worker
		go worker.loop()
	}
	return workers
}

func (worker *lazySMPWorker) loop() {
	for {
		select {
		case work := <-worker.start:
			worker.abort.Store(false)
			worker.thread.searchRoot(lazySMPSearchDepth(work.depth, worker.thread.idx), work.alpha, work.beta, &worker.abort)
			worker.done <- struct{}{}
		case <-worker.stop:
			return
		}
	}
}

func lazySMPStopWorkers(workers []*lazySMPWorker) {
	for _, worker := range workers {
		worker.abort.Store(true)
		close(worker.stop)
	}
}

func lazySMPRunAspirationIteration(rootPos *PositionNG, threads []*lazySMPThreadData, workers []*lazySMPWorker, rootMoves []LazySMPRootMove, depth uint8, alpha, beta Value) {
	for _, thread := range threads {
		thread.prepareRootSearch(rootPos, rootMoves)
	}

	for _, worker := range workers {
		worker.start <- lazySMPWork{
			depth: depth,
			alpha: alpha,
			beta:  beta,
		}
	}

	threads[0].searchRoot(lazySMPSearchDepth(depth, 0), alpha, beta)

	for _, worker := range workers {
		worker.abort.Store(true)
	}
	for _, worker := range workers {
		<-worker.done
	}
}

func (thread *lazySMPThreadData) prepareRootSearch(rootPos *PositionNG, rootMoves []LazySMPRootMove) {
	if thread.pos != rootPos {
		lazySMPSyncRootPosition(thread.pos, rootPos, thread.ctx)
	}
	thread.setRootMoves(rootMoves)
}

func lazySMPSearchDepth(depth uint8, threadIdx int) uint8 {
	return depth
}

func (thread *lazySMPThreadData) searchRoot(depth uint8, alpha, beta Value, abort ...*atomic.Bool) {
	if len(thread.rootMoves) == 0 {
		return
	}

	for i := range thread.rootMoves {
		thread.rootMoves[i].Searched = false
		thread.rootMoves[i].Depth = depth
	}
	thread.searchedRoots = 0
	thread.completedDepth = 0
	thread.aborted = false
	thread.pv.clear()

	if depth == 0 {
		return
	}

	bestScore := -VALUE_INFINITE
	bestMoveIdx := -1
	var abortFlag *atomic.Bool
	if len(abort) > 0 {
		abortFlag = abort[0]
	}

	for i := range thread.rootMoves {
		if abortFlag != nil && abortFlag.Load() {
			thread.aborted = true
			break
		}

		move := thread.rootMoves[i].Move
		thread.pos.DoMove(move, searchState(thread.ctx, thread.pos))

		thread.pv.clear()
		score := -negamaxABWithPVAbort(thread.ctx, -beta, -alpha, thread.pos, depth-1, true, &thread.pv, abortFlag)
		thread.pos.UndoMove(move)

		if abortFlag != nil && abortFlag.Load() {
			thread.aborted = true
			break
		}

		thread.rootMoves[i].Score = score
		lazySMPBuildPVFromSearch(&thread.rootMoves[i], move, &thread.pv)
		thread.rootMoves[i].Searched = true
		thread.searchedRoots++

		if score > bestScore {
			bestScore = score
			bestMoveIdx = i
		}
		if score > alpha {
			alpha = score
			if score >= beta {
				break
			}
		}
	}

	if bestMoveIdx > 0 {
		best := thread.rootMoves[bestMoveIdx]
		copy(thread.rootMoves[1:bestMoveIdx+1], thread.rootMoves[:bestMoveIdx])
		thread.rootMoves[0] = best
	}

	lazySMPSortRootMoves(thread.rootMoves)
	if thread.searchedRoots == len(thread.rootMoves) {
		thread.completedDepth = depth
	}
}

func lazySMPGenerateRootMoves(pos *PositionNG, ctx *SearchContext) []LazySMPRootMove {
	rootMoves := make([]LazySMPRootMove, 0, MAX_MOVES)
	mp := orderMovesByHistory(pos, ctx)
	for move := nextLegalOrderedMove(pos, &mp); move != MOVE_NONE; move = nextLegalOrderedMove(pos, &mp) {
		rootMoves = append(rootMoves, LazySMPRootMove{
			Move:  move,
			Score: -VALUE_INFINITE,
		})
	}
	return rootMoves
}

func (thread *lazySMPThreadData) setRootMoves(rootMoves []LazySMPRootMove) {
	if cap(thread.rootMoves) < len(rootMoves) {
		thread.rootMoves = make([]LazySMPRootMove, len(rootMoves))
	}
	thread.rootMoves = thread.rootMoves[:len(rootMoves)]
	if len(rootMoves) == 0 {
		return
	}

	offset := lazySMPRootMoveOffset(len(rootMoves), thread.idx)
	for i := range thread.rootMoves {
		src := rootMoves[(i+offset)%len(rootMoves)]
		thread.rootMoves[i] = LazySMPRootMove{
			Move:  src.Move,
			Score: src.Score,
			Depth: src.Depth,
		}
	}
}

func lazySMPRootMoveOffset(rootMoveCount, threadIdx int) int {
	if threadIdx == 0 || rootMoveCount < 2 {
		return 0
	}
	offset := threadIdx % rootMoveCount
	if offset == 0 {
		offset = rootMoveCount / 2
	}
	return offset
}

func lazySMPSortRootMoves(rootMoves []LazySMPRootMove) {
	for i := 1; i < len(rootMoves); i++ {
		move := rootMoves[i]
		j := i - 1
		for j >= 0 && lazySMPRootMoveLess(move, rootMoves[j]) {
			rootMoves[j+1] = rootMoves[j]
			j--
		}
		rootMoves[j+1] = move
	}
}

func lazySMPRootMoveLess(left, right LazySMPRootMove) bool {
	if left.Searched != right.Searched {
		return left.Searched
	}
	return left.Score > right.Score
}

func lazySMPBuildPVFromSearch(root *LazySMPRootMove, rootMove MoveNG, searchLine *searchPV) {
	clear(root.PV[:])
	root.PVLength = 0

	if !IsOKMove(rootMove) {
		return
	}

	root.PV[0] = rootMove
	root.PVLength = 1
	if searchLine == nil {
		return
	}

	for ply := 1; ply < searchLine.length[1] && root.PVLength < int(MAX_MOVES); ply++ {
		move := searchLine.table[int(MAX_MOVES)+ply]
		if !IsOKMove(move) {
			break
		}
		root.PV[root.PVLength] = move
		root.PVLength++
	}
}

func lazySMPCopyPosition(pos *PositionNG, ctx *SearchContext) *PositionNG {
	copied := &PositionNG{}
	copyPositionBranchInto(copied, pos, ctx)
	return copied
}

func lazySMPSyncRootPosition(dst, src *PositionNG, ctx *SearchContext) {
	copyPositionBranchInto(dst, src, ctx)
}

func lazySMPPublishMainPV(thread *lazySMPThreadData) {
	if thread == nil || !thread.hasBestRootMove() {
		return
	}

	clear(PvTable[:])
	clear(PvLength[:])
	for i := 0; i < thread.rootMoves[0].PVLength; i++ {
		if i >= len(PvTable) || i >= int(MAX_MOVES) {
			break
		}
		PvTable[i] = thread.rootMoves[0].PV[i]
		PvLength[0] = i + 1
	}
}

func (thread *lazySMPThreadData) hasBestRootMove() bool {
	return thread != nil &&
		len(thread.rootMoves) > 0 &&
		thread.rootMoves[0].Searched &&
		IsOKMove(thread.rootMoves[0].Move)
}

func (thread *lazySMPThreadData) canVote() bool {
	return thread.hasBestRootMove() &&
		!thread.aborted &&
		thread.completedDepth > 0 &&
		thread.searchedRoots == len(thread.rootMoves)
}

func lazySMPVoteBestThread(threads []*lazySMPThreadData) *lazySMPThreadData {
	worstScore := VALUE_INFINITE
	voterCount := 0

	for _, thread := range threads {
		if !thread.canVote() {
			continue
		}
		voterCount++
		worstScore = min(worstScore, thread.rootMoves[0].Score)
	}
	if voterCount == 0 {
		return nil
	}

	var voteMap [SQUARE_NB * SQUARE_NB]Value
	for _, thread := range threads {
		if !thread.canVote() {
			continue
		}
		voteMap[lazySMPFromTo(thread.rootMoves[0].Move)] += lazySMPThreadValue(thread, worstScore)
	}

	bestThread := (*lazySMPThreadData)(nil)
	for _, thread := range threads {
		if thread.canVote() {
			bestThread = thread
			break
		}
	}
	bestScore := bestThread.rootMoves[0].Score
	bestVoteScore := voteMap[lazySMPFromTo(bestThread.rootMoves[0].Move)]

	for _, curr := range threads {
		if curr == bestThread || !curr.canVote() {
			continue
		}
		currScore := curr.rootMoves[0].Score
		currVoteScore := voteMap[lazySMPFromTo(curr.rootMoves[0].Move)]

		if abs(int(bestScore)) >= int(lazySMPTBWinBound) {
			if currScore > bestScore {
				bestThread = curr
				bestScore = currScore
				bestVoteScore = currVoteScore
			}
			continue
		}

		if currScore > -lazySMPTBWinBound &&
			(currVoteScore > bestVoteScore ||
				(currVoteScore == bestVoteScore &&
					lazySMPPVWeightedValue(curr, worstScore) > lazySMPPVWeightedValue(bestThread, worstScore))) {
			bestThread = curr
			bestScore = currScore
			bestVoteScore = currVoteScore
		}
	}

	bestThread.previousScore = bestThread.rootMoves[0].Score
	return bestThread
}

func lazySMPThreadValue(thread *lazySMPThreadData, worstScore Value) Value {
	score := thread.rootMoves[0].Score - worstScore
	if score < 0 {
		score = 0
	}

	depthBonus := Value(thread.completedDepth) * Value(thread.completedDepth)
	return score + depthBonus
}

func lazySMPPVWeightedValue(thread *lazySMPThreadData, worstScore Value) Value {
	if thread.rootMoves[0].PVLength <= 2 {
		return 0
	}
	return lazySMPThreadValue(thread, worstScore)
}

func lazySMPFromTo(move MoveNG) int {
	return FromSQ(move)*SQUARE_NB + ToSQ(move)
}

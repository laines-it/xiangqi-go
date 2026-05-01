package engine

import "unsafe"

const defaultMHTMegabytes = 4

type MinorHashEntry struct {
	Checksum      Key
	Score         Value
	Phase         Phase
	AdvisorNumR   uint8
	AdvisorNumB   uint8
	ElephantNumR  uint8
	ElephantNumB  uint8
	PawnNumR      uint8
	PawnNumB      uint8
	FatalPawnNumR uint8
	FatalPawnNumB uint8
	KingDoorR     uint8
	KingDoorB     uint8
}

type MinorHashTable struct {
	Entries []MinorHashEntry
	Mask    uint64
}

var MHT = NewMinorHashTable(defaultMHTMegabytes)

func NewMinorHashTable(megabytes int) *MinorHashTable {
	megabytes = min(4096, max(1, megabytes))
	size := 1024 * 1024 * megabytes / int(unsafe.Sizeof(MinorHashEntry{}))
	tableSize := roundPowerOfTwo(size)
	return &MinorHashTable{
		Entries: make([]MinorHashEntry, tableSize),
		Mask:    uint64(tableSize - 1),
	}
}

func MHTClear() {
	clear(MHT.Entries)
}

func ProbeMHT(checksum Key, phase Phase) (MinorHashEntry, bool) {
	entry := MHT.Entries[checksum&MHT.Mask]
	if entry.Checksum == checksum && entry.Phase == phase {
		return entry, true
	}
	return MinorHashEntry{}, false
}

func StoreMHT(entry MinorHashEntry) {
	MHT.Entries[entry.Checksum&MHT.Mask] = entry
}

func isMajorPieceType(pt PieceType) bool {
	return pt == ROOK || pt == CANNON || pt == KNIGHT
}

func isMinorPieceType(pt PieceType) bool {
	return pt == KING || pt == PAWN || pt == BISHOP || pt == ADVISOR
}

func (pos *PositionNG) MinorHash() Key {
	return pos.St.Top().minorKey
}

func (pos *PositionNG) MinorPhase() Phase {
	pawnCount := pos.PieceCount[W_PAWN] + pos.PieceCount[B_PAWN]
	defenderCount := pos.PieceCount[W_ADVISOR] + pos.PieceCount[B_ADVISOR] +
		pos.PieceCount[W_BISHOP] + pos.PieceCount[B_BISHOP]
	if pawnCount <= 2 || pawnCount+defenderCount <= 4 {
		return PHASE_ENDGAME
	}
	return PHASE_MIDGAME
}

func (pos *PositionNG) probeMinorEval() Value {
	checksum := pos.MinorHash()
	phase := pos.MinorPhase()
	if entry, ok := ProbeMHT(checksum, phase); ok {
		return entry.Score
	}

	entry := pos.computeMinorHashEntry(checksum, phase)
	StoreMHT(entry)
	return entry.Score
}

func (pos *PositionNG) computeMinorHashEntry(checksum Key, phase Phase) MinorHashEntry {
	entry := MinorHashEntry{
		Checksum:     checksum,
		Phase:        phase,
		AdvisorNumR:  uint8(pos.PieceCount[W_ADVISOR]),
		AdvisorNumB:  uint8(pos.PieceCount[B_ADVISOR]),
		ElephantNumR: uint8(pos.PieceCount[W_BISHOP]),
		ElephantNumB: uint8(pos.PieceCount[B_BISHOP]),
		PawnNumR:     uint8(pos.PieceCount[W_PAWN]),
		PawnNumB:     uint8(pos.PieceCount[B_PAWN]),
	}

	entry.FatalPawnNumR = pos.fatalPawnCount(WHITE)
	entry.FatalPawnNumB = pos.fatalPawnCount(BLACK)
	entry.KingDoorR = pos.kingDoorMask(WHITE)
	entry.KingDoorB = pos.kingDoorMask(BLACK)
	entry.Score = pos.computeMinorEval(entry)

	return entry
}

func (pos *PositionNG) computeMinorEval(entry MinorHashEntry) Value {
	var whiteScore Value
	var blackScore Value

	for sq := Square(0); sq < SQUARE_NB; sq++ {
		pc := pos.Board[sq]
		if pc == NO_PIECE || !isMinorPieceType(TypeOf(pc)) {
			continue
		}

		value := pieceSquareValue(pc, sq)
		if ColorOf(pc) == WHITE {
			whiteScore += value
		} else {
			blackScore += value
		}
	}

	whiteScore += Value(entry.FatalPawnNumR) * 8
	blackScore += Value(entry.FatalPawnNumB) * 8
	whiteScore += Value(bitCount8(entry.KingDoorR)) * 2
	blackScore += Value(bitCount8(entry.KingDoorB)) * 2

	return whiteScore - blackScore
}

func (pos *PositionNG) ComputeDynamicEval() Value {
	var whiteScore Value
	var blackScore Value
	occupied := pos.PiecesAllColor(ALL_PIECES)

	for sq := Square(0); sq < SQUARE_NB; sq++ {
		pc := pos.Board[sq]
		if pc == NO_PIECE || !isMajorPieceType(TypeOf(pc)) {
			continue
		}

		value := pieceSquareValue(pc, sq) + pos.majorActivityBonus(pc, sq, occupied)
		if ColorOf(pc) == WHITE {
			whiteScore += value
		} else {
			blackScore += value
		}
	}

	return whiteScore - blackScore + advancedValue
}

func pieceSquareValue(pc Piece, sq Square) Value {
	idx := sq
	if ColorOf(pc) == BLACK {
		idx = flipSquare(sq)
	}
	return pieceSquareTable[TypeOf(pc)][idx]
}

func (pos *PositionNG) majorActivityBonus(pc Piece, sq Square, occupied Bitboard) Value {
	color := ColorOf(pc)
	attacks := AttacksBB(TypeOf(pc), sq, occupied).And(pos.Pieces(color).Not())
	mobility := Value(attacks.PopCount()) * majorMobilityWeight(TypeOf(pc))
	captures := Value(attacks.And(pos.Pieces(notColor(color))).PopCount()) * 4
	kingPressure := Value(0)
	if attacks.And(pos.Pieces(notColor(color), KING)).IsNotZero() {
		kingPressure = 12
	}
	return mobility + captures + kingPressure
}

func majorMobilityWeight(pt PieceType) Value {
	switch pt {
	case ROOK:
		return 2
	case CANNON:
		return 2
	case KNIGHT:
		return 3
	default:
		return 0
	}
}

func (pos *PositionNG) fatalPawnCount(color Color) uint8 {
	count := uint8(0)
	for pawns := pos.Pieces(color, PAWN); pawns != (Bitboard{}); {
		sq := PopLsb(&pawns)
		if isFatalPawn(color, sq) {
			count++
		}
	}
	return count
}

func isFatalPawn(color Color, sq Square) bool {
	file := FileOf(sq)
	rank := RankOf(sq)
	if file < FILE_D || file > FILE_F {
		return false
	}
	if color == WHITE {
		return rank >= RANK_7
	}
	return rank <= RANK_2
}

var kingDoorSquares = [COLOR_NB][7]Square{
	{SQ_D0, SQ_E0, SQ_F0, SQ_D1, SQ_E1, SQ_F1, SQ_E2},
	{SQ_D9, SQ_E9, SQ_F9, SQ_D8, SQ_E8, SQ_F8, SQ_E7},
}

func (pos *PositionNG) kingDoorMask(color Color) uint8 {
	mask := uint8(0)
	for i, sq := range kingDoorSquares[color] {
		pc := pos.PieceOn(sq)
		if pc != NO_PIECE && ColorOf(pc) == color && isMinorPieceType(TypeOf(pc)) {
			mask |= 1 << i
		}
	}
	return mask
}

func bitCount8(v uint8) int {
	count := 0
	for v != 0 {
		v &= v - 1
		count++
	}
	return count
}

package engine

import (
	"math/rand"
	"testing"
)

func TestMagicRookMatchesSlowGenerator(t *testing.T) {
	for _, occupied := range attackTestOccupancies() {
		for sq := SQ_A0; sq <= SQ_I9; sq++ {
			slow := SlidingAttack(sq, occupied, ROOK)
			fast := AttacksBB(ROOK, sq, occupied)
			if fast != slow {
				t.Fatalf("rook attacks mismatch at sq %d occupied=%#v\nfast=%#v\nslow=%#v", sq, occupied, fast, slow)
			}
		}
	}
}

func TestMagicCannonMatchesSlowGenerator(t *testing.T) {
	for _, occupied := range attackTestOccupancies() {
		for sq := SQ_A0; sq <= SQ_I9; sq++ {
			slowCaptures := SlidingAttack(sq, occupied, CANNON)
			fastCaptures := AttacksBB(CANNON, sq, occupied)
			if fastCaptures != slowCaptures {
				t.Fatalf("cannon capture ray mismatch at sq %d occupied=%#v\nfast=%#v\nslow=%#v", sq, occupied, fastCaptures, slowCaptures)
			}

			slowQuiets := SlidingAttack(sq, occupied, ROOK).And(occupied.Not())
			fastQuiets := AttacksBB(ROOK, sq, occupied).And(occupied.Not())
			if fastQuiets != slowQuiets {
				t.Fatalf("cannon quiet ray mismatch at sq %d occupied=%#v\nfast=%#v\nslow=%#v", sq, occupied, fastQuiets, slowQuiets)
			}
		}
	}
}

func TestKnightTableMatchesSlowGenerator(t *testing.T) {
	for _, occupied := range attackTestOccupancies() {
		for sq := SQ_A0; sq <= SQ_I9; sq++ {
			slow := LameLeaperAttack(KNIGHT, sq, occupied)
			fast := AttacksBB(KNIGHT, sq, occupied)
			if fast != slow {
				t.Fatalf("knight attacks mismatch at sq %d occupied=%#v\nfast=%#v\nslow=%#v", sq, occupied, fast, slow)
			}
		}
	}
}

func TestElephantTableMatchesSlowGenerator(t *testing.T) {
	for _, occupied := range attackTestOccupancies() {
		for sq := SQ_A0; sq <= SQ_I9; sq++ {
			slow := LameLeaperAttack(BISHOP, sq, occupied)
			fast := AttacksBB(BISHOP, sq, occupied)
			if fast != slow {
				t.Fatalf("elephant attacks mismatch at sq %d occupied=%#v\nfast=%#v\nslow=%#v", sq, occupied, fast, slow)
			}
		}
	}
}

func BenchmarkGenerateMovesSlow(b *testing.B) {
	var pos PositionNG
	pos.Set(initialFen)
	var moves [MAX_MOVES]MoveNG
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		negamaxBenchmarkScoreSink = Value(slowGeneratePseudoMoves(&pos, moves[:]))
	}
}

func BenchmarkGenerateMovesMagic(b *testing.B) {
	var pos PositionNG
	pos.Set(initialFen)
	var moves [MAX_MOVES]MoveNG
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		negamaxBenchmarkScoreSink = Value(pos.Generate(PSEUDO_LEGAL, moves[:]))
	}
}

func BenchmarkGenerateLegalMoves(b *testing.B) {
	var pos PositionNG
	pos.Set(initialFen)
	var moves [MAX_MOVES]MoveNG
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		negamaxBenchmarkScoreSink = Value(pos.GenerateLEGAL(moves[:]))
	}
}

func attackTestOccupancies() []Bitboard {
	var pos PositionNG
	pos.Set(initialFen)

	occupancies := []Bitboard{
		{},
		pos.PiecesAllColor(ALL_PIECES),
	}

	rng := rand.New(rand.NewSource(20260508))
	for i := 0; i < 16; i++ {
		occupancies = append(occupancies, randomBoardOccupancy(rng))
	}
	return occupancies
}

func randomBoardOccupancy(rng *rand.Rand) Bitboard {
	return Bitboard{
		Lo: rng.Uint64(),
		Hi: rng.Uint64() & ((uint64(1) << 26) - 1),
	}
}

func slowGeneratePseudoMoves(pos *PositionNG, moves []MoveNG) int {
	us := pos.SideToMove
	occupied := pos.PiecesAllColor(ALL_PIECES)
	empty := occupied.Not()
	enemies := pos.Pieces(notColor(us))
	friendlies := pos.Pieces(us)
	total := 0

	for bb := pos.Pieces(us); bb != (Bitboard{}); {
		from := PopLsb(&bb)
		pc := pos.PieceOn(from)
		var attacks Bitboard
		switch TypeOf(pc) {
		case PAWN:
			attacks = PawnAttacks[us][from].And(friendlies.Not())
		case ROOK:
			attacks = SlidingAttack(from, occupied, ROOK).And(friendlies.Not())
		case CANNON:
			quiet := SlidingAttack(from, occupied, ROOK).And(empty)
			captures := SlidingAttack(from, occupied, CANNON).And(enemies)
			attacks = quiet.Or(captures)
		case KNIGHT:
			attacks = LameLeaperAttack(KNIGHT, from, occupied).And(friendlies.Not())
		case BISHOP:
			attacks = LameLeaperAttack(BISHOP, from, occupied).And(friendlies.Not())
		case ADVISOR, KING:
			attacks = AttacksBBEmptyOcc(TypeOf(pc), from).And(friendlies.Not())
		}
		for attacks != (Bitboard{}) {
			moves[total] = MakeMove(from, PopLsb(&attacks))
			total++
		}
	}

	return total
}

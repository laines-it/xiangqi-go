package engine

import (
	"fmt"
	"testing"
)

var demoSink Bitboard

func squareName(s Square) string {
	return fmt.Sprintf("%c%d", 'a'+FileOf(s), RankOf(s))
}

func bitboardSquares(b Bitboard) []string {
	var out []string
	for s := SQ_A0; s <= SQ_I9; s++ {
		if b.And(SquareBB[s]).IsNotZero() {
			out = append(out, squareName(s))
		}
	}
	return out
}

func TestMagicDemo(t *testing.T) {
	sq := SQ_B3
	occupied := SquareBB[SQ_B6].Or(SquareBB[SQ_E3]).Or(SquareBB[SQ_H8])
	m := &RookMagics[sq]
	relevant := occupied.And(m.mask)
	attack := AttacksBB(ROOK, sq, occupied)
	sequential := SlidingAttack(sq, occupied, ROOK)

	t.Logf("square=%s", squareName(sq))
	t.Logf("mask_popcount=%d shift=%d table_variants=%d", m.mask.PopCount(), m.shift, 1<<m.mask.PopCount())
	t.Logf("magic_hi=0x%016X magic_lo=0x%016X", m.magic.Hi, m.magic.Lo)
	t.Logf("occupied=%v", bitboardSquares(occupied))
	t.Logf("relevant_occupied=%v", bitboardSquares(relevant))
	t.Logf("index=%d", m.Index(occupied))
	t.Logf("magic_attack=%v", bitboardSquares(attack))
	t.Logf("sequential_attack=%v equal=%t", bitboardSquares(sequential), attack == sequential)

	cm := &CannonMagics[sq]
	cannonAttack := AttacksBB(CANNON, sq, occupied)
	cannonSequential := SlidingAttack(sq, occupied, CANNON)
	t.Logf("cannon_index=%d", cm.Index(occupied))
	t.Logf("cannon_attack=%v equal=%t", bitboardSquares(cannonAttack), cannonAttack == cannonSequential)
}

func BenchmarkMagicVsSlidingDemo(b *testing.B) {
	occupied := SquareBB[SQ_B6].Or(SquareBB[SQ_E3]).Or(SquareBB[SQ_H8])

	b.Run("MagicRook", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			demoSink = AttacksBB(ROOK, SQ_B3, occupied)
		}
	})

	b.Run("SequentialRook", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			demoSink = SlidingAttack(SQ_B3, occupied, ROOK)
		}
	})

	b.Run("MagicCannon", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			demoSink = AttacksBB(CANNON, SQ_B3, occupied)
		}
	})

	b.Run("SequentialCannon", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			demoSink = SlidingAttack(SQ_B3, occupied, CANNON)
		}
	})
}

func BenchmarkAllPieceTypeAttacksDemo(b *testing.B) {
	occupied := SquareBB[SQ_B6].
		Or(SquareBB[SQ_E3]).
		Or(SquareBB[SQ_H8]).
		Or(SquareBB[SQ_D3]).
		Or(SquareBB[SQ_C5])

	cases := []struct {
		name   string
		attack func() Bitboard
	}{
		{"RookMagic", func() Bitboard { return AttacksBB(ROOK, SQ_B3, occupied) }},
		{"AdvisorPseudo", func() Bitboard { return AttacksBB(ADVISOR, SQ_E1, occupied) }},
		{"CannonMagic", func() Bitboard { return AttacksBB(CANNON, SQ_B3, occupied) }},
		{"PawnTable", func() Bitboard { return PawnAttacks[WHITE][SQ_E6] }},
		{"KnightMagic", func() Bitboard { return AttacksBB(KNIGHT, SQ_B3, occupied) }},
		{"BishopMagic", func() Bitboard { return AttacksBB(BISHOP, SQ_C2, occupied) }},
		{"KingPseudo", func() Bitboard { return AttacksBB(KING, SQ_E1, occupied) }},
		{"KnightToMagic", func() Bitboard { return AttacksBB(KNIGHT_TO, SQ_B3, occupied) }},
	}

	b.ReportAllocs()
	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				demoSink = tc.attack()
			}
		})
	}
}

func BenchmarkMagicVsComputedDemo(b *testing.B) {
	occupied := SquareBB[SQ_B6].
		Or(SquareBB[SQ_E3]).
		Or(SquareBB[SQ_H8]).
		Or(SquareBB[SQ_D3]).
		Or(SquareBB[SQ_C5])

	cases := []struct {
		name     string
		magic    func() Bitboard
		computed func() Bitboard
	}{
		{
			name:     "Rook",
			magic:    func() Bitboard { return AttacksBB(ROOK, SQ_B3, occupied) },
			computed: func() Bitboard { return SlidingAttack(SQ_B3, occupied, ROOK) },
		},
		{
			name:     "Cannon",
			magic:    func() Bitboard { return AttacksBB(CANNON, SQ_B3, occupied) },
			computed: func() Bitboard { return SlidingAttack(SQ_B3, occupied, CANNON) },
		},
		{
			name:     "Knight",
			magic:    func() Bitboard { return AttacksBB(KNIGHT, SQ_B3, occupied) },
			computed: func() Bitboard { return LameLeaperAttack(KNIGHT, SQ_B3, occupied) },
		},
		{
			name:     "Bishop",
			magic:    func() Bitboard { return AttacksBB(BISHOP, SQ_C2, occupied) },
			computed: func() Bitboard { return LameLeaperAttack(BISHOP, SQ_C2, occupied) },
		},
		{
			name:     "KnightTo",
			magic:    func() Bitboard { return AttacksBB(KNIGHT_TO, SQ_B3, occupied) },
			computed: func() Bitboard { return LameLeaperAttack(KNIGHT_TO, SQ_B3, occupied) },
		},
	}

	b.ReportAllocs()
	for _, tc := range cases {
		tc := tc
		b.Run(tc.name+"/Magic", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				demoSink = tc.magic()
			}
		})
		b.Run(tc.name+"/Computed", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				demoSink = tc.computed()
			}
		})
	}
}

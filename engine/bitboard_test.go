package engine

import "testing"

func TestBitboardSetClearHasBoundaries(t *testing.T) {
	var bb Bitboard
	for _, sq := range []Square{0, 63, 64, 89} {
		bb.Set(sq)
		if !bb.Has(sq) {
			t.Fatalf("expected square %d to be set", sq)
		}
	}

	if bb.Lo != (uint64(1) | uint64(1)<<63) {
		t.Fatalf("Lo mismatch: got %#x", bb.Lo)
	}
	if bb.Hi != (uint64(1) | uint64(1)<<25) {
		t.Fatalf("Hi mismatch: got %#x", bb.Hi)
	}
	if bb.PopCount() != 4 {
		t.Fatalf("PopCount=%d, want 4", bb.PopCount())
	}

	for _, sq := range []Square{0, 63, 64, 89} {
		bb.Clear(sq)
		if bb.Has(sq) {
			t.Fatalf("expected square %d to be clear", sq)
		}
	}
	if !bb.IsZero() {
		t.Fatalf("bitboard should be empty after clearing all squares: %#v", bb)
	}
}

func TestBitboardSetClearBelowAndAbove64(t *testing.T) {
	var low Bitboard
	low.Set(12)
	low.Set(63)
	if low.Hi != 0 || !low.Has(12) || !low.Has(63) {
		t.Fatalf("low squares should live only in Lo: %#v", low)
	}
	low.Clear(12)
	if low.Has(12) || !low.Has(63) {
		t.Fatalf("clearing one low square corrupted another: %#v", low)
	}

	var high Bitboard
	high.Set(64)
	high.Set(88)
	if high.Lo != 0 || !high.Has(64) || !high.Has(88) {
		t.Fatalf("high squares should live only in Hi: %#v", high)
	}
	high.Clear(64)
	if high.Has(64) || !high.Has(88) {
		t.Fatalf("clearing one high square corrupted another: %#v", high)
	}
}

func TestBitboardPopCountAndIteration(t *testing.T) {
	var bb Bitboard
	want := []Square{1, 9, 64, 70, 89}
	for _, sq := range want {
		bb.Set(sq)
	}

	if got := bb.PopCount(); got != uint(len(want)) {
		t.Fatalf("PopCount=%d, want %d", got, len(want))
	}

	for i, sq := range want {
		if got := bb.PopLSB(); got != sq {
			t.Fatalf("iteration[%d]=%d, want %d", i, got, sq)
		}
	}
	if !bb.IsZero() {
		t.Fatalf("iterator should have consumed all bits: %#v", bb)
	}
}

func TestBitboardBooleanOperations(t *testing.T) {
	var a Bitboard
	var b Bitboard
	a.Set(0)
	a.Set(64)
	b.Set(64)
	b.Set(89)

	if got := a.Or(b).PopCount(); got != 3 {
		t.Fatalf("Or popcount=%d, want 3", got)
	}
	if got := a.And(b); got.PopCount() != 1 || !got.Has(64) {
		t.Fatalf("And mismatch: %#v", got)
	}
	if got := a.Xor(b); got.PopCount() != 2 || !got.Has(0) || !got.Has(89) {
		t.Fatalf("Xor mismatch: %#v", got)
	}
	if got := a.AndNot(b); got.PopCount() != 1 || !got.Has(0) {
		t.Fatalf("AndNot mismatch: %#v", got)
	}
}

package engine

import (
	"fmt"
	"strings"
)

var fenPieceChars = [PIECE_NB]byte{
	W_ROOK:    'R',
	W_ADVISOR: 'A',
	W_CANNON:  'C',
	W_PAWN:    'P',
	W_KNIGHT:  'N',
	W_BISHOP:  'B',
	W_KING:    'K',
	B_ROOK:    'r',
	B_ADVISOR: 'a',
	B_CANNON:  'c',
	B_PAWN:    'p',
	B_KNIGHT:  'n',
	B_BISHOP:  'b',
	B_KING:    'k',
}

// FEN returns the current position as a Xiangqi FEN string.
func (pos *PositionNG) FEN() string {
	var b strings.Builder

	for r := RANK_9; r >= RANK_0; r-- {
		empty := 0
		for f := FILE_A; f <= FILE_I; f++ {
			pc := pos.PieceOn(MakeSquareNG(f, r))
			if pc == NO_PIECE {
				empty++
				continue
			}
			if empty > 0 {
				b.WriteByte(byte('0' + empty))
				empty = 0
			}
			b.WriteByte(fenPieceChars[pc])
		}
		if empty > 0 {
			b.WriteByte(byte('0' + empty))
		}
		if r > RANK_0 {
			b.WriteByte('/')
		}
	}

	side := "w"
	if pos.SideToMove == BLACK {
		side = "b"
	}

	rule60 := 0
	if len(pos.St) > 0 {
		rule60 = pos.St.Top().Rule60
	}
	fullMove := pos.GamePly/2 + 1

	return fmt.Sprintf("%s %s - - %d %d", b.String(), side, rule60, fullMove)
}

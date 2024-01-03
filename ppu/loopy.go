package ppu

import (
	"fmt"
)

// loopy struct will store v and t (loopy registers) and allow
// extracting and setting the various components as described below:
// yyy NN YYYYY XXXXX
// ||| || ||||| +++++-- coarse X scroll
// ||| || +++++-------- coarse Y scroll
// ||| ++-------------- nametable select
// +++----------------- fine Y scroll
type loopy uint16

func (l *loopy) String() string {
	return fmt.Sprintf("%03b:%01b%01b:%05b:%05b", l.fineY(), l.nametableY(), l.nametableX(), l.coarseY(), l.coarseX())
}

func (l *loopy) set(n uint16) {
	*l = loopy(n)
}

func (l *loopy) resetCoarseX() {
	*l &= 0xFFE0
}

func (l *loopy) coarseX() uint16 {
	return uint16(*l & 0x001F)
}

func (l *loopy) setCoarseX(n uint16) {
	*l = *l&0xFFE0 | loopy(n)
}

func (l *loopy) incrementCoarseX() {
	*l += 1
}

func (l *loopy) coarseY() uint16 {
	return uint16(*l&0x03E0) >> 5
}

func (l *loopy) incrementCoarseY() {
	*l += 32
}

func (l *loopy) resetCoarseY() {
	*l &= 0xFC1F
}

func (l *loopy) setCoarseY(n uint16) {
	*l = *l&0xFC1F | loopy(n<<5)
}

func (l *loopy) nametableX() uint16 {
	return uint16(*l) & 0x0400 >> 10
}

func (l *loopy) setNametableX(val uint8) {
	*l = *l&0xFBFF | loopy((uint16(val&0x01) << 10))
}

func (l *loopy) toggleNametableX() {
	*l ^= 0x0400
}

func (l *loopy) nametableY() uint16 {
	return uint16(*l) & 0x0800 >> 11
}

func (l *loopy) toggleNametableY() {
	*l ^= 0x0800
}

func (l *loopy) setNametableY(val uint8) {
	*l = *l&0xF7FF | loopy((uint16(val&0x1) << 11))
}

func (l *loopy) fineY() uint16 {
	return uint16(*l) & 0x7000 >> 12
}

func (l *loopy) incrementFineY() {
	*l += 0x1000 // 4096; 1 << 12
}

func (l *loopy) setFineY(n uint16) {
	*l = loopy(uint16(*l) & (0x0FFF | (uint16(n) << 12)))
}

func (l *loopy) resetFineY() {
	*l &= 0x0FFF
}

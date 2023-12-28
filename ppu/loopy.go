package ppu

// loopy struct will store v and t (loopy registers) and allow
// extracting and setting the various components as described below:
// yyy NN YYYYY XXXXX
// ||| || ||||| +++++-- coarse X scroll
// ||| || +++++-------- coarse Y scroll
// ||| ++-------------- nametable select
// +++----------------- fine Y scroll
type loopy struct {
	data uint16 // only 15 bits used
}

func (l *loopy) coarseX() uint16 {
	return l.data & 0x001F
}

func (l *loopy) setCoarseX(n uint16) {
	l.data = (l.data & 0xFFE0) | n
}

func (l *loopy) incrementCoarseX() {
	l.data += 1
}

func (l *loopy) coarseY() uint16 {
	return (l.data & 0x03E0) >> 5
}

func (l *loopy) incrementCoarseY() {
	l.data = ((l.coarseY() + 1) << 5) | (l.data & 0xFC1F)
}

func (l *loopy) setCoarseY(n uint16) {
	l.data = (l.data & 0xFC1F) | (uint16(n) << 5)
}

func (l *loopy) nametableX() uint16 {
	return (l.data & 0x0400) >> 10
}

func clearBit(n, pos uint16) uint16 {
	return n &^ (uint16(1) << (pos - 1))
}

func (l *loopy) toggleNametableX() {
	if l.nametableX() == 1 {
		l.data = clearBit(l.data, 11)
	} else {
		l.data |= (uint16(1) << 10)
	}
}

func (l *loopy) nametableY() uint16 {
	return (l.data & 0x0800) >> 11
}

func (l *loopy) toggleNametableY() {
	if l.nametableY() == 1 {
		l.data = clearBit(l.data, 12)
	} else {
		l.data |= (uint16(1) << 11)
	}
}

func (l *loopy) fineY() uint16 {
	return (l.data & 0x7000) >> 12
}

func (l *loopy) incrementFineY() {
	l.data = (l.data & 0x0FFF) | ((l.fineY() + 1) << 12)
}

func (l *loopy) setFineY(n uint16) {
	l.data &= 0x0FFF | (uint16(n) << 12)
}

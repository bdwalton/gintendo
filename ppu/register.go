package ppu

type addrReg struct {
	high, low uint8
	lowB      bool // true if we're writing the low byte, false if writing high byte
}

func (ar *addrReg) get() uint16 {
	return (uint16(ar.high) << 8) | uint16(ar.low)
}

func (ar *addrReg) set(val uint8) {
	switch ar.lowB {
	case true:
		ar.low = val
	default:
		ar.high = val
	}

	ar.lowB = !ar.lowB
}

func (ar *addrReg) reset() {
	ar.low, ar.high = 0, 0
	ar.lowB = false
}

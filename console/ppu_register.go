package console

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

// 7  bit  0
// ---- ----
// VPHB SINN
// |||| ||||
// |||| ||++- Base nametable address
// |||| ||    (0 = $2000; 1 = $2400; 2 = $2800; 3 = $2C00)
// |||| |+--- VRAM address increment per CPU read/write of PPUDATA
// |||| |     (0: add 1, going across; 1: add 32, going down)
// |||| +---- Sprite pattern table address for 8x8 sprites
// ||||       (0: $0000; 1: $1000; ignored in 8x16 mode)
// |||+------ Background pattern table address (0: $0000; 1: $1000)
// ||+------- Sprite size (0: 8x8 pixels; 1: 8x16 pixels)
// |+-------- PPU master/slave select
// |          (0: read backdrop from EXT pins; 1: output color on EXT pins)
// +--------- Generate an NMI at the start of the vertical blanking interval (0: off; 1: on)
type ctrlReg struct {
	val uint8
}

func (cr *ctrlReg) set(val uint8) {
	cr.val = val
}

const (
	CTRL_NAMETABLE1             = 1
	CTRL_NAMETABLE2             = 1 << 1
	CTRL_VRAM_ADD_INCREMENT     = 1 << 2
	CTRL_SPRITE_PATTERN_ADDR    = 1 << 3
	CTRL_BACKROUND_PATTERN_ADDR = 1 << 4
	CTRL_SPRITE_SIZE            = 1 << 5
	CTRL_MASTER_SLAVE_SELECT    = 1 << 6
	CTRL_GENERATE_NMI           = 1 << 7

	CTRL_INCR_ACROSS = 1
	CTRL_INCR_DOWN   = 32
)

func (cr *ctrlReg) vram_increment() uint8 {
	if cr.val&CTRL_VRAM_ADD_INCREMENT > 1 {
		return CTRL_INCR_DOWN
	}

	return CTRL_INCR_ACROSS
}

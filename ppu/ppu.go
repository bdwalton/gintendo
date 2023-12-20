// Package ppu implements the PPU hardward in the NES
package ppu

const (
	VRAM_SIZE    = 2048
	OAM_SIZE     = 256
	PALETTE_SIZE = 32
)

// Special Registers
const (
	PPUCTRL   = 0x2000
	PPUMASK   = 0x2001
	PPUSTATUS = 0x2002
	OAMADDR   = 0x2003
	OAMDATA   = 0x2004
	PPUSCROLL = 0x2005
	PPUADDR   = 0x2006
	PPUDATA   = 0x2007
	OAMDMA    = 0x4014
)

type Bus interface {
	ChrRead(uint16) uint8
	TriggerNMI()
}

type PPU struct {
	bus          Bus
	paletteTable [PALETTE_SIZE]uint8
	oamData      [OAM_SIZE]uint8
	vram         [VRAM_SIZE]uint8
	ppuAddr      *addrReg
	mirrorMode   uint8
	registers    map[uint16]uint8
}

func New(b Bus) *PPU {
	return &PPU{
		bus:       b,
		ppuAddr:   &addrReg{},
		registers: make(map[uint16]uint8),
	}
}

func (p *PPU) WriteReg(r uint16, val uint8) {
	switch r {
	case PPUADDR:
		p.ppuAddr.set(val)
	default:
		p.registers[r] = val
	}
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

// readReg returns the current value of a register.
func (p *PPU) ReadReg(r uint16) uint8 {
	return p.registers[r]
}

func (p *PPU) vram_increment() uint8 {
	if p.ReadReg(PPUCTRL)&CTRL_VRAM_ADD_INCREMENT > 1 {
		return CTRL_INCR_DOWN
	}

	return CTRL_INCR_ACROSS
}

type addrReg struct {
	high, low uint8
	lowB      bool // true if we're writing the low byte, false if writing high byte
}

func (ar *addrReg) get16() uint16 {
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

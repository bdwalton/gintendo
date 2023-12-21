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

// PPUCTRL bit flags
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
// +--------- Generate an NMI at the start of the
//
//	vertical blanking interval (0: off; 1: on)
const (
	CTRL_NAMETABLE1             = 1
	CTRL_NAMETABLE2             = 1 << 1
	CTRL_VRAM_ADD_INCREMENT     = 1 << 2
	CTRL_SPRITE_PATTERN_ADDR    = 1 << 3
	CTRL_BACKROUND_PATTERN_ADDR = 1 << 4
	CTRL_SPRITE_SIZE            = 1 << 5
	CTRL_MASTER_SLAVE_SELECT    = 1 << 6
	CTRL_GENERATE_NMI           = 1 << 7
)

// VRAM increment options
const (
	CTRL_INCR_ACROSS = 1
	CTRL_INCR_DOWN   = 32
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
	mirrorMode   uint8
	// The memory mapped registered that the CPU can read/write
	// from. PPUADDR is special because it needs to handle 2
	// writes to form a 16-bit address
	ppuAddr   *addrReg
	registers map[uint16]uint8
	// internal registers
	v, t uint16 // current vram addr, temp vram addr; only 15 bits used
	x    uint8  // fine x scroll, only 3 bits used
	w    uint8  // first or second write toggle; 1 bit

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

	switch r {
	case PPUCTRL:
		p.t = (p.t & 0xF3FF) | (uint16(val&0x03) << 10)
	case PPUSCROLL:
		if p.w == 0 {
			p.t = (p.t & 0xFFE0) | (uint16(val&0xF8) >> 3)
			p.x = (val & 0x07)
			p.w = 1
		} else {

			p.t = (uint16(val)&0x0007)<<12 | (p.t & 0x0C00) | (uint16(val)&0x00F8)<<2 | (p.t & 0x001F)
			p.w = 0
		}
	case PPUADDR:
		if p.w == 0 {
			p.t = (p.t & 0b10111111_11111111) | (uint16(val&0x3F) << 8)
			p.w = 1
		} else {
			p.t = (p.t & 0xFF00) | uint16(val)
			p.v = p.t
			p.w = 0
		}
	}
}

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

// Mirroring mode
const (
	MIRROR_HORIZONTAL = iota
	MIRROR_VERTICAL
	MIRROR_FOUR_SCREEN
)

const (
	PATTERN_TABLE_0  = 0x0000
	PATTERN_TABLE_1  = 0x1000
	NAMETABLE_0      = 0x2000
	NAMETABLE_1      = 0x2400
	NAMETABLE_2      = 0x2800
	NAMETABLE_3      = 0x2C00
	NAMETABLE_MIRROR = 0x3EFF
	PALETTE_RAM      = 0x3F00
	PALETTE_MIRROR   = 0x3F20
)

func (p *PPU) tileMapAddr(addr uint16) uint16 {
	// Now we have a as the base of our internal memory
	a := addr - NAMETABLE_0
	// https://www.nesdev.org/wiki/Mirroring#Nametable_Mirroring
	switch p.mirrorMode {
	case MIRROR_FOUR_SCREEN:
		panic("we don't have mapper support to leverage vram on catridge")
	case MIRROR_HORIZONTAL:
		if a >= 0x800 {
			return 0x0400 + ((a - 0x800) % 0x400)
		}
		return a % 0x0400
	case MIRROR_VERTICAL:
		return a % 0x800
	}

	panic("unkown mirroring mode")
}

func (p *PPU) read(addr uint16) uint8 {
	a := addr % 0x4000

	switch {
	case a < NAMETABLE_0:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		return p.bus.ChrRead(a)
	case a < PALETTE_RAM:
		return p.vram[p.tileMapAddr(a)]
	case a < NAMETABLE_MIRROR:
		return p.vram[p.tileMapAddr(a-NAMETABLE_0)]
	case a < PALETTE_MIRROR:
		return p.vram[a-PALETTE_RAM]
	default:
		x := (a - PALETTE_RAM) % 0x0020
		return p.vram[PALETTE_RAM+x]
	}
}

func (p *PPU) generateNMI() bool {
	return p.registers[PPUCTRL]&CTRL_GENERATE_NMI > 0
}

// Tick executes n cycles. We call it tick instead of step because
// there is no real logic. It's just a fixed loop in the hardware.
func (p *PPU) Tick(n int) {
	if p.generateNMI() {
		p.bus.TriggerNMI()
	}

	for i := 0; i < n; i++ {
		p.tick()
	}
}

// This is the main execution logic for the PPU
func (p *PPU) tick() {

}

// Package ppu implements the PPU hardward in the NES
package ppu

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

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
	screen       *sdl.Surface
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

func New(b Bus, w *sdl.Window) (*PPU, error) {
	s, err := w.GetSurface()
	if err != nil {
		return nil, fmt.Errorf("couldn't obtain SDL surface: %v", err)
	}
	return &PPU{
		bus:       b,
		screen:    s,
		ppuAddr:   &addrReg{},
		registers: make(map[uint16]uint8),
	}, nil
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
	switch r {
	case PPUSTATUS:
		p.w = 0
	}

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

func newColor(r, g, b uint8) sdl.Color {
	return sdl.Color{R: r, G: g, B: b}
}

var SYSTEM_PALETTE [64]sdl.Color = [64]sdl.Color{
	newColor(0x80, 0x80, 0x80), newColor(0x00, 0x3D, 0xA6), newColor(0x00, 0x12, 0xB0), newColor(0x44, 0x00, 0x96), newColor(0xA1, 0x00, 0x5E),
	newColor(0xC7, 0x00, 0x28), newColor(0xBA, 0x06, 0x00), newColor(0x8C, 0x17, 0x00), newColor(0x5C, 0x2F, 0x00), newColor(0x10, 0x45, 0x00),
	newColor(0x05, 0x4A, 0x00), newColor(0x00, 0x47, 0x2E), newColor(0x00, 0x41, 0x66), newColor(0x00, 0x00, 0x00), newColor(0x05, 0x05, 0x05),
	newColor(0x05, 0x05, 0x05), newColor(0xC7, 0xC7, 0xC7), newColor(0x00, 0x77, 0xFF), newColor(0x21, 0x55, 0xFF), newColor(0x82, 0x37, 0xFA),
	newColor(0xEB, 0x2F, 0xB5), newColor(0xFF, 0x29, 0x50), newColor(0xFF, 0x22, 0x00), newColor(0xD6, 0x32, 0x00), newColor(0xC4, 0x62, 0x00),
	newColor(0x35, 0x80, 0x00), newColor(0x05, 0x8F, 0x00), newColor(0x00, 0x8A, 0x55), newColor(0x00, 0x99, 0xCC), newColor(0x21, 0x21, 0x21),
	newColor(0x09, 0x09, 0x09), newColor(0x09, 0x09, 0x09), newColor(0xFF, 0xFF, 0xFF), newColor(0x0F, 0xD7, 0xFF), newColor(0x69, 0xA2, 0xFF),
	newColor(0xD4, 0x80, 0xFF), newColor(0xFF, 0x45, 0xF3), newColor(0xFF, 0x61, 0x8B), newColor(0xFF, 0x88, 0x33), newColor(0xFF, 0x9C, 0x12),
	newColor(0xFA, 0xBC, 0x20), newColor(0x9F, 0xE3, 0x0E), newColor(0x2B, 0xF0, 0x35), newColor(0x0C, 0xF0, 0xA4), newColor(0x05, 0xFB, 0xFF),
	newColor(0x5E, 0x5E, 0x5E), newColor(0x0D, 0x0D, 0x0D), newColor(0x0D, 0x0D, 0x0D), newColor(0xFF, 0xFF, 0xFF), newColor(0xA6, 0xFC, 0xFF),
	newColor(0xB3, 0xEC, 0xFF), newColor(0xDA, 0xAB, 0xEB), newColor(0xFF, 0xA8, 0xF9), newColor(0xFF, 0xAB, 0xB3), newColor(0xFF, 0xD2, 0xB0),
	newColor(0xFF, 0xEF, 0xA6), newColor(0xFF, 0xF7, 0x9C), newColor(0xD7, 0xE8, 0x95), newColor(0xA6, 0xED, 0xAF), newColor(0xA2, 0xF2, 0xDA),
	newColor(0x99, 0xFF, 0xFC), newColor(0xDD, 0xDD, 0xDD), newColor(0x11, 0x11, 0x11), newColor(0x11, 0x11, 0x11),
}

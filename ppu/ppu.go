// Package ppu implements the PPU hardward in the NES
package ppu

import (
	"fmt"
	"image/color"
)

const (
	VRAM_SIZE    = 2048
	OAM_SIZE     = 256
	PALETTE_SIZE = 32
)

// Display constants
const (
	NES_RES_WIDTH  = 256
	NES_RES_HEIGHT = 240
)

// Special Registers. These are the addresses on which they're exposed
// to the CPU. When we get calls to WriteReg from the Bus that's
// driving us, we'll get these values because that's all the CPU knows
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

// 7  bit  0
// ---- ----
// VSO. ....
// |||| ||||
// |||+-++++- PPU open bus. Returns stale PPU bus contents.
// ||+------- Sprite overflow. The intent was for this flag to be set
// ||         whenever more than eight sprites appear on a scanline, but a
// ||         hardware bug causes the actual behavior to be more complicated
// ||         and generate false positives as well as false negatives; see
// ||         PPU sprite evaluation. This flag is set during sprite
// ||         evaluation and cleared at dot 1 (the second dot) of the
// ||         pre-render line.
// |+-------- Sprite 0 Hit.  Set when a nonzero pixel of sprite 0 overlaps
// |          a nonzero background pixel; cleared at dot 1 of the pre-render
// |          line.  Used for raster timing.
// +--------- Vertical blank has started (0: not in vblank; 1: in vblank).
//
//	Set at dot 1 of line 241 (the line *after* the post-render
//	line); cleared after reading $2002 and at dot 1 of the
//	pre-render line.
const (
	STATUS_SPRITE_OVERFLOW = 1 << 5
	STATUS_SPRITE_0_HIT    = 1 << 6
	STATUS_VERTICAL_BLANK  = 1 << 7
)

// 7  bit  0
// ---- ----
// BGRs bMmG
// |||| ||||
// |||| |||+- Greyscale (0: normal color, 1: produce a greyscale display)
// |||| ||+-- 1: Show background in leftmost 8 pixels of screen, 0: Hide
// |||| |+--- 1: Show sprites in leftmost 8 pixels of screen, 0: Hide
// |||| +---- 1: Show background
// |||+------ 1: Show sprites
// ||+------- Emphasize red (green on PAL/Dendy)
// |+-------- Emphasize green (red on PAL/Dendy)
// +--------- Emphasize blue

// Mask flags
const (
	MASK_GREYSCALE         = 1 << 0
	MASK_SHOW_LEFT_TILES   = 1 << 1
	MASK_SHOW_LEFT_SPRITES = 1 << 2
	MASK_RENDER_BG         = 1 << 3
	MASK_RENDER_FG         = 1 << 4
	MASK_EMPHASIZE_RED     = 1 << 5
	MASK_EMPHASIZE_GREEN   = 1 << 6
	MASK_EMPHASIZE_BLUE    = 1 << 7
)

type Bus interface {
	ChrRead(uint16, uint16) []uint8
	TriggerNMI()
	MirrorMode() uint8
}

type PPU struct {
	bus          Bus
	ticks        int64
	pixels       []color.RGBA
	paletteTable [PALETTE_SIZE]uint8
	oamData      [OAM_SIZE]uint8
	vram         [VRAM_SIZE]uint8
	mirrorMode   uint8

	// internal registers
	v, t   loopy // current vram addr, temp vram addr
	x      uint8 // fine x scroll, only 3 bits used
	wLatch uint8 // first or second write toggle; 1 bit

	// registers that maintain state not captured in v, t, etc.
	ctrl   uint8
	status uint8
	mask   uint8

	scanline int16 // -1 through 261 (0 - 239 are visible)
	scandot  int16 // 0 through 320 (1 - 256 are visible)

	// For reads from registers that are delayed due to cycle counts
	bufferData uint8
}

func New(b Bus) *PPU {
	ps := NES_RES_WIDTH * NES_RES_HEIGHT
	px := make([]color.RGBA, ps, ps)
	for i := 0; i < ps; i++ {
		px[i] = color.RGBA{0, 0, 0, 0xff} // Black
	}
	return &PPU{
		bus:        b,
		pixels:     px,
		mirrorMode: b.MirrorMode(),
	}
}

func (p *PPU) String() string {
	return fmt.Sprintf("x=%d, y=%d, v=%s fineX=%03b (t=%s), ctrl=%08b,mask=%08b,status=%08b ", p.scandot, p.scanline, p.v.String(), p.x, p.t.String(), p.ctrl, p.mask, p.status)
}

func (p *PPU) GetPixels() []color.RGBA {
	return p.pixels
}

func (p *PPU) GetResolution() (int, int) {
	return NES_RES_WIDTH, NES_RES_HEIGHT
}

func (p *PPU) WriteReg(r uint16, val uint8) {
	switch r {
	case PPUCTRL:
		p.ctrl = val
		// we set loopy t's nametable x and y
		p.t.setNametableX(val)
		p.t.setNametableY(val >> 1)
	case PPUMASK:
		p.mask = val
	case PPUSCROLL:
		if p.wLatch == 0 {
			p.t.setCoarseX(uint16(val) >> 3)
			p.x = (val & 0x07)
			p.wLatch = 1
		} else {
			// we set loopy t's coarse y and fine y
			p.t.setCoarseY(uint16(val) & 0x00F8 >> 3)
			p.t.setFineY(uint16(val) & 0x0007)
			p.wLatch = 0
		}
	case PPUADDR:
		if p.wLatch == 0 {
			p.t.set((uint16(val&0x3F) << 8) | (uint16(p.t) & 0x00FF))
			p.wLatch = 1
		} else {
			p.t.set((uint16(p.t) & 0xFF00) | uint16(val))
			p.v.set(uint16(p.t))
			p.wLatch = 0
		}
	case PPUDATA:
		p.write(uint16(p.v), val)
		p.vramIncrement()
	}
}

// ReadReg returns the current value of a register.
func (p *PPU) ReadReg(r uint16) uint8 {
	var ret uint8 = 0x00 // Most regstiers aren't readable, so we'll return 0
	switch r {
	case PPUSTATUS:
		// From NESDev - we fill the status register with the
		// bottom contents of the buffered data.
		ret = (p.status & 0xE0) | (p.bufferData & 0x1F)
		p.clearVBlank()
		p.wLatch = 0
	case PPUDATA:
		ret = p.bufferData
		p.bufferData = p.read(uint16(p.v))
		// When reading from palette range, we don't suffer
		// the cycle delay that we do when reading other data.
		if p.v > 0x3F00 {
			ret = p.bufferData
		}
		p.vramIncrement()
	}

	return ret
}

func (p *PPU) vramIncrement() {
	x := uint16(CTRL_INCR_ACROSS)
	if p.ctrl&CTRL_VRAM_ADD_INCREMENT > 0 {
		x = CTRL_INCR_DOWN
	}

	p.v = loopy(uint16(p.v) + x)
}

// Mirroring mode
const (
	MIRROR_HORIZONTAL = iota
	MIRROR_VERTICAL
	MIRROR_FOUR_SCREEN
)

const (
	PATTERN_TABLE_0      = 0x0000
	PATTERN_TABLE_1      = 0x1000
	BASE_NAMETABLE       = 0x2000
	ATTRIBUTE_OFFSET     = 0x03C0 // each nametable has attribute data at the end of it
	NAMETABLE_0          = BASE_NAMETABLE
	NAMETABLE_1          = 0x2400
	NAMETABLE_2          = 0x2800
	NAMETABLE_3          = 0x2C00
	NAMETABLE_END        = 0x2FFF
	NAMETABLE_MIRROR     = 0x3000
	NAMETABLE_MIRROR_END = 0x3EFF
	PALETTE_RAM          = 0x3F00
	PALETTE_MIRROR       = 0x3F20
)

// tileMapAddr handles mirror mode mapping of addresses with the
// 0x2000-0x2FFF. It takes the natural address and returns the mapped
// address within the same range.
func (p *PPU) tileMapAddr(addr uint16) uint16 {
	a := addr - BASE_NAMETABLE
	// https://www.nesdev.org/wiki/Mirroring#Nametable_Mirroring
	switch p.mirrorMode {
	case MIRROR_FOUR_SCREEN:
		panic("we don't have mapper support to leverage vram on catridge")
	case MIRROR_HORIZONTAL:
		if a >= 0x800 {
			a = 0x400 + ((a - 0x800) % 0x400)
		} else {
			a %= 0x0400
		}
	case MIRROR_VERTICAL:
		a %= 0x800
	}

	return a + BASE_NAMETABLE
}

func (p *PPU) read(addr uint16) uint8 {
	// 0x4000 - 0xFFFF is mirrored to 0x0000 - 0x3FFF
	a := addr % 0x4000

	switch {
	case a < NAMETABLE_0:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		return p.bus.ChrRead(a, a)[0]
	case a < PALETTE_RAM:
		return p.vram[BASE_NAMETABLE-p.tileMapAddr(a)]
	case a < NAMETABLE_MIRROR:
		return p.vram[BASE_NAMETABLE-p.tileMapAddr(a-NAMETABLE_0)]
	case a < PALETTE_MIRROR:
		return p.vram[a-PALETTE_RAM]
	default:
		x := (a - PALETTE_RAM) % 0x0020
		return p.vram[PALETTE_RAM+x]
	}
}

func (p *PPU) write(addr uint16, val uint8) {
	a := addr % 0x4000

	switch {
	case a < NAMETABLE_0:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		panic("we don't support writting to CHR ROM")
	case a < PALETTE_RAM:
		p.vram[p.tileMapAddr(a)] = val
	case a < NAMETABLE_MIRROR:
		p.vram[p.tileMapAddr(a-NAMETABLE_0)] = val
	case a < PALETTE_MIRROR:
		p.vram[a-PALETTE_RAM] = val
	default:
		x := (a - PALETTE_RAM) % 0x0020
		p.vram[PALETTE_RAM+x] = val
	}
}

func (p *PPU) generateNMI() bool {
	return p.ctrl&CTRL_GENERATE_NMI > 0
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

func (p *PPU) clearVBlank() {
	p.status &^= STATUS_VERTICAL_BLANK
}

func (p *PPU) setVBlank() {
	p.status |= STATUS_VERTICAL_BLANK
}

// This is the main execution logic for the PPU
func (p *PPU) tick() {
	// Do real work here
	if p.scanline >= -1 && p.scanline < 240 {
		if p.scanline == -1 && p.scandot == 1 {
			p.clearVBlank()
		}
	}
	p.ticks += 1

	bank := 0

	for tile_n := 0; tile_n < 32; tile_n++ {
		s := uint16(bank + (tile_n * 16))
		e := uint16(s+15) + 1
		tile := p.bus.ChrRead(s, e)

		for y := 0; y < 8; y++ {
			u, l := tile[y], tile[y+8]
			for x := 7; x > 0; x-- {
				val := (1&u)<<1 | (1 & l)
				u >>= 1
				l >>= 1

				posx := x + (tile_n*8)%NES_RES_WIDTH
				posy := (y * NES_RES_WIDTH)
				p.pixels[posy+posx] = SYSTEM_PALETTE[val]
			}
		}
	}
}

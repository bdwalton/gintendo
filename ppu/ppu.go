// Package ppu implements the PPU hardward in the NES
package ppu

import (
	"fmt"
	"image"
	"image/color"
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
// +--------- Generate an NMI at the start of the vertical blanking
//
//	interval (0: off; 1: on)
const (
	CTRL_NAMETABLE1              = 1
	CTRL_NAMETABLE2              = 1 << 1
	CTRL_VRAM_ADD_INCREMENT      = 1 << 2
	CTRL_SPRITE_PATTERN_ADDR     = 1 << 3
	CTRL_BACKGROUND_PATTERN_ADDR = 1 << 4
	CTRL_SPRITE_SIZE             = 1 << 5
	CTRL_MASTER_SLAVE_SELECT     = 1 << 6
	CTRL_GENERATE_NMI            = 1 << 7
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
	ChrRead(uint16) uint8
	TriggerNMI()
	MirrorMode() uint8
}

type PPU struct {
	bus          Bus
	pixels       *image.RGBA
	paletteTable [32]uint8
	oamData      [256]uint8
	vram         [2048]uint8 // 2k of video ram
	mirrorMode   uint8

	// internal registers
	v, t   loopy // current vram addr, temp vram addr
	x      uint8 // fine x scroll, only 3 bits used
	wLatch uint8 // first or second write toggle; 1 bit

	// registers that maintain state not captured in v, t, etc.
	ctrl   uint8
	status uint8
	mask   uint8

	scanline uint16 // 0 through 261 (0 - 239 are visible)
	scandot  uint16 // 0 through 320 (1 - 256 are visible)
	frame    uint64
	oddFrame bool

	// For reads from registers that are delayed due to cycle counts
	bufferData uint8

	// rendering variables for the background
	bgNextTileID     uint8  // next bg tile id
	bgNextTileAttrib uint8  // next bg tile attribute
	bgNextTileLsb    uint8  // next bg tile least significant byte
	bgNextTileMsb    uint8  // next bg tile most significant byte
	bgSPLo           uint16 // shift pattern low
	bgSPHi           uint16 // shift pattern high
	bgSALo           uint16 // shift attribute low
	bgSAHi           uint16 // shift attribute low
}

func New(b Bus) *PPU {
	ps := NES_RES_WIDTH * NES_RES_HEIGHT
	px := make([]color.RGBA, ps, ps)
	for i := 0; i < ps; i++ {
		px[i] = color.RGBA{0, 0, 0, 0xFF} // Black
	}

	ppu := &PPU{
		bus:        b,
		pixels:     image.NewRGBA(image.Rect(0, 0, NES_RES_WIDTH, NES_RES_HEIGHT)),
		mirrorMode: b.MirrorMode(),
	}
	ppu.Reset()

	return ppu
}

func (p *PPU) Reset() {
	p.scandot = 340
	p.scanline = 240
	p.frame = 0
	p.oddFrame = false
	p.ctrl = 0
	p.mask = 0
	p.status = 0
}

func (p *PPU) String() string {
	return fmt.Sprintf("x=%d, y=%d, v=%s fineX=%03b (t=%s), ctrl=%08b,mask=%08b,status=%08b ", p.scandot, p.scanline, p.v.String(), p.x, p.t.String(), p.ctrl, p.mask, p.status)
}

func (p *PPU) GetPixels() *image.RGBA {
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
	switch p.ctrl * CTRL_VRAM_ADD_INCREMENT >> 2 {
	case 0:
		p.v.incrementCoarseX() // Move across (== p.v++)
	case 1:
		p.v.incrementCoarseY() // Move down (== p.v+=32)
	}
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
	NAMETABLE_END        = 0x2FFF
	NAMETABLE_MIRROR_END = 0x3EFF
	PALETTE_RAM          = 0x3F00
	PALETTE_MIRROR_END   = 0x3FFF
)

// tileMapAddr handles mirror mode mapping of addresses with the
// 0x2000-0x2FFF. It takes the natural address and returns the mapped
// address within the vram range (2k).
func (p *PPU) tileMapAddr(addr uint16) uint16 {
	a := addr & 0x0FFF
	// https://www.nesdev.org/wiki/Mirroring#Nametable_Mirroring
	switch p.mirrorMode {
	case MIRROR_FOUR_SCREEN:
		panic("we don't have mapper support to leverage vram on catridge")
	case MIRROR_VERTICAL:
		switch {
		case (a >= 0 && a <= 0x03FF) || (a >= 0x0800 && a <= 0x0BFF): // table 0
			a &= 0x03FF
		case (a >= 0x0400 && a <= 0x07FF) || (a >= 0x0C00 && a <= 0x0FFF): // table 1
			a = (a & 0x03FF) + 0x400
		}
	case MIRROR_HORIZONTAL:
		switch {
		case (a >= 0 && a <= 0x07FF): // table 0
			a &= 0x03FF
		case (a >= 0x0800 && a <= 0x0FFF): // table 1
			a = (a & 0x03FF) + 0x400
		}
	}

	return a
}

// Address range  Size   Description
// $0000-$0FFF    $1000  Pattern table 0
// $1000-$1FFF	  $1000  Pattern table 1
// $2000-$23FF	  $0400  Nametable 0
// $2400-$27FF	  $0400  Nametable 1
// $2800-$2BFF	  $0400  Nametable 2
// $2C00-$2FFF	  $0400  Nametable 3
// $3000-$3EFF	  $0F00  Mirrors of $2000-$2EFF
// $3F00-$3F1F	  $0020  Palette RAM indexes
// $3F20-$3FFF	  $00E0  Mirrors of $3F00-$3F1F

func (p *PPU) read(addr uint16) uint8 {
	// 0x4000 - 0xFFFF is mirrored to 0x0000 - 0x3FFF
	a := addr & 0x3FFF

	switch {
	case a < BASE_NAMETABLE:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		return p.bus.ChrRead(a)
	case a <= NAMETABLE_MIRROR_END:
		return p.vram[p.tileMapAddr((a&0x0FFF)+BASE_NAMETABLE)]
	case a >= PALETTE_RAM && a <= PALETTE_MIRROR_END: // Palette Table
		a &= 0x001F // handle mirroring
		switch a {
		case 0x0010:
			a = 0x0000
		case 0x0014:
			a = 0x0004
		case 0x0018:
			a = 0x0008
		case 0x001C:
			a = 0x000C
		}
		val := p.paletteTable[a]
		switch p.mask & MASK_GREYSCALE {
		case 0:
			val &= 0x3F
		case 1:
			val &= 0x30
		}

		return val
	}

	panic("Shouldn't be reached")
	return 0
}

func (p *PPU) write(addr uint16, val uint8) {
	// 0x4000 - 0xFFFF is mirrored to 0x0000 - 0x3FFF
	a := addr & 0x3FFF

	switch {
	case a < BASE_NAMETABLE:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		// TODO(bdwalton): Add write support
	case a <= NAMETABLE_MIRROR_END:
		p.vram[p.tileMapAddr((a&0x0FFF)+BASE_NAMETABLE)] = val
	case a >= PALETTE_RAM && a <= PALETTE_MIRROR_END: // Palette Table
		a &= 0x001F // handle mirroring
		switch a {
		case 0x0010:
			a = 0x0000
		case 0x0014:
			a = 0x0004
		case 0x0018:
			a = 0x0008
		case 0x001C:
			a = 0x000C
		}
		p.paletteTable[a] = val
	}
}

func (p *PPU) generateNMI() bool {
	return p.ctrl&CTRL_GENERATE_NMI > 0
}

func (p *PPU) clearVBlank() {
	p.status &^= STATUS_VERTICAL_BLANK
}

func (p *PPU) setVBlank() {
	p.status |= STATUS_VERTICAL_BLANK
}

func (p *PPU) nmiEnabled() bool {
	return p.ctrl&CTRL_GENERATE_NMI > 0
}

func (p *PPU) renderBackground() bool {
	return p.mask&MASK_RENDER_BG > 0
}

func (p *PPU) renderForeground() bool {
	return p.mask&MASK_RENDER_FG > 0
}

func (p *PPU) renderingEnabled() bool {
	return p.renderBackground() || p.renderForeground()
}

func (p *PPU) loadBGShifters() {
	p.bgSPLo = (p.bgSPLo & 0xFF00) | uint16(p.bgNextTileLsb)
	p.bgSPHi = (p.bgSPHi & 0xFF00) | uint16(p.bgNextTileMsb)

	p.bgSALo = p.bgSALo & 0xFF00
	if p.bgNextTileAttrib&0x01 == 1 {
		p.bgSALo |= 0xFF
	}

	p.bgSAHi = p.bgSAHi & 0xFF00
	if p.bgNextTileAttrib&0x02 > 0 {
		p.bgSAHi |= 0xFF
	}
}

func (p *PPU) backgroundTableID() uint8 {
	return p.ctrl & CTRL_BACKGROUND_PATTERN_ADDR >> 4
}

func (p *PPU) visibleLine() bool {
	return p.scanline >= 0 && p.scanline < 240
}

func (p *PPU) visibleDot() bool {
	return p.scandot >= 1 && p.scandot <= 256
}

func (p *PPU) prerenderLine() bool {
	return p.scanline == 261
}

func (p *PPU) renderLine() bool {
	return p.visibleLine() || p.prerenderLine()
}

func (p *PPU) prefetchCycle() bool {
	return p.scandot >= 321 && p.scandot <= 336
}

func (p *PPU) fetchCycle() bool {
	return p.visibleDot() || p.prefetchCycle()
}

func (p *PPU) incrementScan() {
	if p.renderingEnabled() && p.oddFrame && p.prerenderLine() && p.scandot == 339 {
		p.scandot = 0
		p.scanline = 0
		p.frame++
		p.oddFrame = !p.oddFrame
		return
	}

	p.scandot++
	if p.scandot >= 341 {
		p.scandot = 0
		p.scanline += 1
		if p.scanline > 261 {
			p.scanline = 0
			p.frame++
			p.oddFrame = !p.oddFrame
		}
	}
}

// Tick executes a PPU cycle. We call it tick instead of step because
// there is no real logic. It's just a fixed loop in the hardware.
// Documented at:
// https://www.nesdev.org/w/images/default/4/4f/Ppu.svg.
func (p *PPU) Tick() {
	p.incrementScan()

	// handle background pixels
	if p.renderingEnabled() {
		if p.visibleLine() && p.visibleDot() { // render the pixel
			var bgPix uint8 = 0x00 // 2 bit pixel to be rendered
			var bgPal uint8 = 0x00 // 3 bit index of the palette used by the pixel
			if p.renderBackground() {
				var mux uint16 = (0x8000 >> p.x)

				var p0Pix, p1Pix uint8

				if p.bgSPLo&mux > 0 {
					p0Pix = 1
				}

				if p.bgSPHi&mux > 0 {
					p1Pix = 1
				}

				bgPix = (p1Pix << 1) | p0Pix

				var bgPal0, bgPal1 uint8
				if p.bgSALo&mux > 0 {
					bgPal0 = 1
				}
				if p.bgSAHi&mux > 0 {
					bgPal1 = 1
				}
				bgPal = (bgPal1 << 1) | bgPal0
			}
			addr := uint16(0x3F00) + (uint16(bgPal) << 2) + uint16(bgPix)
			p.pixels.Set(int(p.scandot-1), int(p.scanline), SYSTEM_PALETTE[p.read(addr)&0x3F])
		}

		if p.renderLine() && p.fetchCycle() {
			if p.renderBackground() {
				p.bgSPLo <<= 1
				p.bgSPHi <<= 1

				p.bgSAHi <<= 1
				p.bgSAHi <<= 1
			}

			switch p.scandot % 8 {
			case 0: // next tile
				// Prime the next 8 pixels worth of tile data
				p.loadBGShifters()
				// The next tile is in the nametable
				// data but only 12 bits of the loopy
				// v register are useful for
				// discovering it.
				p.bgNextTileID = p.read(BASE_NAMETABLE | (uint16(p.v) & 0x0FFF))
			case 1: // nametable byte
				// Increment X scroll, but only when rendering is enabled
				// https://www.nesdev.org/wiki/PPU_scrolling
				if p.renderingEnabled() {
					if p.v.coarseX() == 31 {
						p.v.resetCoarseX()
						p.v.toggleNametableX()
					} else {
						p.v.incrementCoarseX()
					}
				}
			case 3: // attribute table byte
				coarseY := p.v.coarseY()
				coarseX := p.v.coarseX()

				a := BASE_NAMETABLE |
					ATTRIBUTE_OFFSET | // All with the attribute space of nametables
					(p.v.nametableY() << 11) | // nametable select
					(p.v.nametableX() << 10) | // nametable select
					((coarseY >> 2) << 3) | // high 3 bits of coarse Y
					(coarseX >> 2) // high 3 bits of coarse X
				p.bgNextTileAttrib = p.read(a)

				if coarseY&0x02 > 0 {
					p.bgNextTileAttrib >>= 4
				}
				if coarseX&0x02 > 0 {
					p.bgNextTileAttrib >>= 2
				}

				p.bgNextTileAttrib &= 0x03
			case 5: // bg tile lsb
				a := (uint16(p.backgroundTableID()) << 12) +
					(uint16(p.bgNextTileID) << 4) +
					uint16(p.v.fineY())
				p.bgNextTileLsb = p.read(a)

			case 7: // bg tile msb
				a := (uint16(p.backgroundTableID()) << 12) +
					(uint16(p.bgNextTileID) << 4) +
					p.v.fineY() +
					8 // Offset by 8 to select next plane
				p.bgNextTileMsb = p.read(a)
			}
		}

		if p.prerenderLine() && p.scandot >= 280 && p.scandot <= 304 {
			// copy Y
			p.v.setFineY(p.t.fineY())
			p.v.setNametableY(uint8(p.t.nametableY()))
			p.v.setCoarseY(p.t.coarseY())
		}

		if p.renderLine() {
			if p.fetchCycle() && p.scandot%8 == 0 {
				// increment X
			}
			if p.scandot == 256 { // increment Y
				// https://www.nesdev.org/wiki/PPU_scrolling
				// Scroll Y, but only if rendering is enabled
				// If possible, just increment the fine y offset
				if p.v.fineY() < 7 {
					p.v.incrementFineY()
				} else {
					p.v.resetFineY()
					switch p.v.coarseY() {
					case 29:
						p.v.resetCoarseY()
						p.v.toggleNametableY()
					case 31:
						p.v.resetCoarseY()
					default:
						p.v.incrementCoarseY()
					}
				}

			}
			if p.scandot == 257 {
				p.loadBGShifters()
				// Only if rendering is enabled
				if p.renderingEnabled() {
					p.v.setNametableX(uint8(p.t.nametableX()))
					p.v.setCoarseX(p.t.coarseX())
				}
			}
		}
	}

	if p.scanline == 241 && p.scandot == 1 {
		p.setVBlank()
		if p.nmiEnabled() {
			p.bus.TriggerNMI()
		}
	}

	if p.prerenderLine() && p.scandot == 1 {
		p.clearVBlank()
	}
}

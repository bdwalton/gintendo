package ppu

type priority uint8

const (
	FRONT priority = iota
	BACK
)

type oam struct {
	// Y position of top of sprite. Sprite data is delayed by one
	// scanline; you must subtract 1 from the sprite's Y
	// coordinate before writing it here. Hide a sprite by moving
	// it down offscreen, by writing any values between #$EF-#$FF
	// here. Sprites are never displayed on the first line of the
	// picture, and it is impossible to place a sprite partially
	// off the top of the screen.
	y uint8
	// For 8x8 sprites, this is the tile number of this sprite
	// within the pattern table selected in bit 3 of PPUCTRL
	// ($2000). For 8x16 sprites (bit 5 of PPUCTRL set), the PPU
	// ignores the pattern table selection and selects a pattern
	// table from bit 0 of this number.
	tileId uint8
	// See above

	palette      uint8
	renderP      priority
	flipV, flipH bool

	// X position of left side of sprite. X-scroll values of
	// $F9-FF results in parts of the sprite to be past the right
	// edge of the screen, thus invisible. It is not possible to
	// have a sprite partially visible on the left edge. Instead,
	// left-clipping through PPUMASK ($2001) can be used to
	// simulate this effect.
	x uint8
}

func OAMFromBytes(in []uint8) oam {
	// 76543210 -> in[2]
	// ||||||||
	// ||||||++- Palette (4 to 7) of sprite
	// |||+++--- Unimplemented (read 0)
	// ||+------ Priority (0: in front of background; 1: behind background)
	// |+------- Flip sprite horizontally
	// +-------- Flip sprite vertically
	return oam{
		y:       in[0],
		tileId:  in[1],
		palette: (in[2] & 0x03),
		renderP: priority((in[2] & 0x20) >> 5),
		flipH:   ((in[2] & 0x40) >> 6) == 1,
		flipV:   ((in[2] & 0x80) >> 7) == 1,
		x:       in[3],
	}
}

func (o oam) attributes() uint8 {
	a := o.palette | uint8(o.renderP<<5)
	if o.flipH {
		a |= (1 << 6)
	}
	if o.flipV {
		a |= (1 << 7)
	}

	return a
}

package console

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Buttons, as bits:
// 0 - A
// 1 - B
// 2 - Select
// 3 - Start
// 4 - Up
// 5 - Down
// 6 - Left
// 7 - Right
var keys []ebiten.Key = []ebiten.Key{
	ebiten.KeyA,     // A
	ebiten.KeyB,     // B
	ebiten.KeySpace, // Select
	ebiten.KeyEnter, // Start
	ebiten.KeyUp,    // Up
	ebiten.KeyDown,  // Down
	ebiten.KeyLeft,  // Left
	ebiten.KeyRight, // Right
}

type controller struct {
	strobe  bool
	buttons uint8
	idx     uint8
}

func (c *controller) write(val uint8) {
	switch val & 0x01 {
	case 0:
		c.strobe = false
		c.buttons = 0
		c.poll()

	case 1:
		c.strobe = true
		c.idx = 0
	}
}

func (c *controller) read() uint8 {
	if c.idx > 7 {
		return 1
	}

	ret := c.buttons & (1 << c.idx) >> c.idx
	c.idx++
	return ret
}

func (c *controller) poll() {
	for i, key := range keys {
		var pressed uint8
		if ebiten.IsKeyPressed(key) {
			pressed = 1
		}
		c.buttons |= (pressed << i)
	}
}

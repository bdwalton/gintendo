package console

import (
	"testing"

	"github.com/bdwalton/gintendo/mappers"
)

func TestBaseNESMapping(t *testing.T) {
	b := New(mappers.Dummy, NES_MODE)
	c := b.cpu

	for i := 0; i < 10; i++ {
		c.Write(uint16(i), uint8(i+1))
	}

	for _, a := range []uint16{0, 0x800, 0x1000, 0x1800} {
		for i := 0; i < 10; i++ {
			if got := c.Read(a + uint16(i)); got != uint8(i+1) {
				t.Errorf("mem[%04x] = %02x, wanted %02x", a, got, i+1)
			}

		}
	}

}

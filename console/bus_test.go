package console

import (
	"testing"

	"github.com/bdwalton/gintendo/mappers"
)

func TestBaseNESMapping(t *testing.T) {
	b := New(mappers.Dummy, NES_MODE)

	for i := 0; i < 10; i++ {
		b.Write(uint16(i), uint8(i+1))
	}

	for _, a := range []uint16{0, 0x800, 0x1000, 0x1800} {
		for i := 0; i < 10; i++ {
			if got := b.Read(a + uint16(i)); got != uint8(i+1) {
				t.Errorf("mem[%04x] = %02x, wanted %02x", a, got, i+1)
			}

		}
	}

}

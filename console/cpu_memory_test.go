package console

import (
	"testing"

	"github.com/bdwalton/gintendo/mappers"
)

func TestBaseMapping(t *testing.T) {
	m := newCPUMemory(RAM_SIZE, mappers.Dummy)

	for i := 0; i < 10; i++ {
		m.write(uint16(i), uint8(i+1))
	}

	for _, a := range []uint16{0, 0x800, 0x1000, 0x1800} {
		for i := 0; i < 10; i++ {
			if got := m.read(a + uint16(i)); got != uint8(i+1) {
				t.Errorf("mem[%04x] = %02x, wanted %02x", a, got, i+1)
			}

		}
	}

}

package console

import (
	"testing"

	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/nesrom"
)

func TestNameTableMirroring(t *testing.T) {
	dm := mappers.Dummy
	m := newPPUMemory(VRAM_SIZE, dm)

	cases := []struct {
		a       uint16 // address to write
		val, mm uint8  // value to write, mirroring mode
		wantAp  uint16 // address to validate for mirroring, in addition to original

	}{
		{0x2000, 0xF1, nesrom.MIRROR_VERTICAL, 0x2800},
		{0x20FF, 0x1F, nesrom.MIRROR_VERTICAL, 0x28FF},
		{0x2801, 0xE3, nesrom.MIRROR_VERTICAL, 0x2001},
		{0x240F, 0xD1, nesrom.MIRROR_VERTICAL, 0x2C0F},
		{0x2C1E, 0xCC, nesrom.MIRROR_VERTICAL, 0x241E},
		{0x2000, 0xF2, nesrom.MIRROR_HORIZONTAL, 0x2400},
		{0x2800, 0x32, nesrom.MIRROR_HORIZONTAL, 0x2C00},
		{0x2C00, 0x41, nesrom.MIRROR_HORIZONTAL, 0x2800},
		{0x2402, 0x56, nesrom.MIRROR_HORIZONTAL, 0x2002},
		{0x2CFF, 0x15, nesrom.MIRROR_HORIZONTAL, 0x28FF},
	}

	for i, tc := range cases {
		dm.MM = tc.mm
		m.write(tc.a, tc.val)
		if got, gotAp := m.read(tc.a), m.read(tc.wantAp); got != tc.val || gotAp != tc.val {
			t.Errorf("%d: %04x: %02x, %04x: %02x, wanted %02x", i, tc.a, got, tc.wantAp, gotAp, tc.val)
		}
	}
}

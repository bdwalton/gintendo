package console

import (
	"testing"
)

func TestAddrReg(t *testing.T) {
	cases := []struct {
		inputs []uint8  // we'll feed bytes...
		wants  []uint16 // and check the value after each
	}{
		{
			[]uint8{0x0F, 0x0B, 0x10, 0x02},
			[]uint16{0x0F00, 0x0F0B, 0x100B, 0x1002},
		},
		{
			[]uint8{0x1F, 0xB0},
			[]uint16{0x1F00, 0x1FB0},
		},
	}

	var ar addrReg
	for i, tc := range cases {
		for j, x := range tc.inputs {
			ar.set(x)
			if got := ar.get(); got != tc.wants[j] {
				t.Errorf("%d: Got %04x, want %04x", i, got, tc.wants[j])
			}
		}
		ar.reset()
	}
}

func TestCtrlRegVRAMIncrement(t *testing.T) {
	cases := []struct {
		val  uint8
		want uint8
	}{
		{CTRL_VRAM_ADD_INCREMENT, CTRL_INCR_DOWN},
		{CTRL_NAMETABLE1 | CTRL_NAMETABLE2, CTRL_INCR_ACROSS},
	}

	cr := &ctrlReg{}
	for i, tc := range cases {
		cr.set(tc.val)
		if got := cr.vram_increment(); got != tc.want {
			t.Errorf("%d: Got %d, want %d", i, got, tc.want)
		}
	}
}

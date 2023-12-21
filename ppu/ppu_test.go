package ppu

import (
	"testing"
)

type testBus struct {
	nmiTriggered bool
}

func (tb *testBus) ChrRead(uint16) uint8 {
	return 0
}

func (tb *testBus) TriggerNMI() {
	tb.nmiTriggered = true
}

func (tb *testBus) reset() {
	tb.nmiTriggered = false
}

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
			if got := ar.get16(); got != tc.wants[j] {
				t.Errorf("%d: Got %04x, want %04x", i, got, tc.wants[j])
			}
		}
		ar.reset()
	}
}

func TestWriteRegPPUCTRL(t *testing.T) {
	cases := []struct {
		val   uint8
		wantT uint16
	}{
		// These are cumulative
		{0b11001100, 0b00000000_00000000},
		{0b01010101, 0b00000100_00000000},
		{0b01010111, 0b00001100_00000000},
		{0b01010100, 0b00000000_00000000},
		{0b01010110, 0b00001000_00000000},
	}

	p := New(&testBus{})
	for i, tc := range cases {
		p.WriteReg(PPUCTRL, tc.val)
		if p.t != tc.wantT {
			t.Errorf("%d: Got t=%015b wanted %015b", i, p.t, tc.wantT)
		}
	}
}

func TestWriteRegPPUSCROLL(t *testing.T) {
	cases := []struct {
		val   uint8
		wantT uint16
		wantX uint8
		wantW uint8
	}{
		// These are cumulative
		{0b11001100, 0b00000000_00011001, 0b00000100, 1},
		{0b01010101, 0b01010001_01011001, 0b00000100, 0},
		{0b11111111, 0b01010001_01011111, 0b00000111, 1},
		{0b00000000, 0b00000000_00011111, 0b00000111, 0},
		{0b01101010, 0b00000000_00001101, 0b00000010, 1},
		{0b01101010, 0b00100001_10101101, 0b00000010, 0},
	}

	p := New(&testBus{})
	for i, tc := range cases {
		p.WriteReg(PPUSCROLL, tc.val)
		if p.t != tc.wantT || p.x != tc.wantX || p.w != tc.wantW {
			t.Errorf("%d: Got t,x,w=%015b,%03b,%d, wanted %015b,%03b,%d", i, p.t, p.x, p.w, tc.wantT, tc.wantX, tc.wantW)
		}
	}
}

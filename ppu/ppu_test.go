package ppu

import (
	"github.com/veandco/go-sdl2/sdl"
	"testing"
)

func init() {
	sdl.Init(sdl.INIT_EVERYTHING)
	window, _ = sdl.CreateWindow("gintendo-test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 256, 240, sdl.WINDOW_HIDDEN)
	sdl.EnableScreenSaver()
}

type testBus struct {
	nmiTriggered bool
}

func (tb *testBus) ChrRead(start, end uint16) []uint8 {
	return []uint8{0}
}

func (tb *testBus) TriggerNMI() {
	tb.nmiTriggered = true
}

func (tb *testBus) reset() {
	tb.nmiTriggered = false
}

var window *sdl.Window

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

	p := New(&testBus{}, window)

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

	p := New(&testBus{}, window)
	for i, tc := range cases {
		p.WriteReg(PPUSCROLL, tc.val)
		if p.t != tc.wantT || p.x != tc.wantX || p.w != tc.wantW {
			t.Errorf("%d: Got t,x,w=%015b,%03b,%d, wanted %015b,%03b,%d", i, p.t, p.x, p.w, tc.wantT, tc.wantX, tc.wantW)
		}
	}
}

func TestWriteRegPPUADDR(t *testing.T) {
	cases := []struct {
		val    uint8
		startT uint16
		wantT  uint16
		wantV  uint16
		wantW  uint8
	}{
		// These are cumulative
		{0b11001100, 0b1000000_00000000, 0b00001100_00000000, 0x0000, 1},
		{0b11001100, 0b00001100_00000000, 0b00001100_11001100, 0b00001100_11001100, 0},
		{0b11111111, 0b00001100_11001100, 0b00111111_11001100, 0b00001100_11001100, 1},
		{0b10001110, 0b00111111_11001100, 0b00111111_10001110, 0b00111111_10001110, 0},
	}

	p := New(&testBus{}, window)

	for i, tc := range cases {
		p.t = tc.startT
		p.WriteReg(PPUADDR, tc.val)
		if p.t != tc.wantT || p.v != tc.wantV || p.w != tc.wantW {
			t.Errorf("%d: Got t,v,w=%015b,%015b,%d,\n\t\t   wanted %015b,%015b,%d", i, p.t, p.v, p.w, tc.wantT, tc.wantV, tc.wantW)
		}
	}
}

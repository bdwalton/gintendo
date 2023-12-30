package ppu

import (
	"testing"
)

type testBus struct {
	nmiTriggered bool
	mirrormode   uint8
}

func (tb *testBus) MirrorMode() uint8 {
	return tb.mirrormode
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

func TestClearVBlank(t *testing.T) {
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x80, 0x00},
		{0x91, 0x11},
	}

	for i, tc := range cases {
		p := New(&testBus{})
		p.status = tc.status
		p.clearVBlank()
		if p.status != tc.want {
			t.Errorf("%d: Got 0x%02x, wanted 0x%02x", i, p.status, tc.want)
		}
	}

}

func TestSetVBlank(t *testing.T) {
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x00, 0x80},
		{0x11, 0x91},
	}

	for i, tc := range cases {
		p := New(&testBus{})
		p.status = tc.status
		p.setVBlank()
		if p.status != tc.want {
			t.Errorf("%d: Got 0x%02x, wanted 0x%02x", i, p.status, tc.want)
		}
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
		if uint16(p.t) != tc.wantT {
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
		{0b01010101, 0b00000001_01011001, 0b00000100, 0},
		{0b11111111, 0b00000001_01011111, 0b00000111, 1},
		{0b00000000, 0b00000000_00011111, 0b00000111, 0},
		{0b01101010, 0b00000000_00001101, 0b00000010, 1},
		{0b01101010, 0b00000001_10101101, 0b00000010, 0},
	}

	p := New(&testBus{})
	for i, tc := range cases {
		p.WriteReg(PPUSCROLL, tc.val)
		if uint16(p.t) != tc.wantT || p.x != tc.wantX || p.wLatch != tc.wantW {
			t.Errorf("%d: Got t,x,w=%015b,%03b,%d, wanted:\n\t\t         %015b,%03b,%d", i, p.t, p.x, p.wLatch, tc.wantT, tc.wantX, tc.wantW)
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

	p := New(&testBus{})

	for i, tc := range cases {
		p.t = loopy(tc.startT)
		p.WriteReg(PPUADDR, tc.val)
		if uint16(p.t) != tc.wantT || uint16(p.v) != tc.wantV || p.wLatch != tc.wantW {
			t.Errorf("%d: Got t,v,w=%015b,%015b,%d,\n\t\t   wanted %015b,%015b,%d", i, p.t, p.v, p.wLatch, tc.wantT, tc.wantV, tc.wantW)
		}
	}
}

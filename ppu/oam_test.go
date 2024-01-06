package ppu

import (
	"testing"
)

func TestOAMAttributes(t *testing.T) {
	cases := []struct {
		attrib         uint8
		wantPa         uint8
		wantPr         priority
		wantFH, wantFV bool
	}{
		{0b11111111, 0x03, BACK, true, true},
		{0b01111111, 0x03, BACK, true, false},
		{0b00111111, 0x03, BACK, false, false},
		{0b00111101, 0x01, BACK, false, false},
		{0b00011101, 0x01, FRONT, false, false},
		{0b10011101, 0x01, FRONT, false, true},
		{0b10011110, 0x02, FRONT, false, true},
	}

	for i, tc := range cases {
		o := OAMFromBytes([]uint8{0, 0, tc.attrib, 0})

		if o.palette != tc.wantPa || o.renderP != tc.wantPr || o.flipH != tc.wantFH || o.flipV != tc.wantFV {
			t.Errorf("%d: %02x, %d, %t, %t; wanted %02x, %d, %t, %t", i, o.palette, o.renderP, o.flipH, o.flipV, tc.wantPa, tc.wantPr, tc.wantFH, tc.wantFV)
		}
	}
}

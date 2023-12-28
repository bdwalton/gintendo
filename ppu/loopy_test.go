package ppu

import (
	"testing"
)

func TestLoopyGet(t *testing.T) {
	cases := []struct {
		data                           uint16
		wantCoarseX, wantCoarseY       uint16
		wantNameTableX, wantNameTableY uint16
		wantFineY                      uint16
	}{
		{0b0000_0000_0000_0000, 0, 0, 0, 0, 0},
		{0b0111_1011_1001_1000, 0b11000, 0b11100, 0, 1, 0b111},
		{0b0011_0111_1001_0111, 0b10111, 0b11100, 1, 0, 0b011},
		{0b0011_1111_1001_0111, 0b10111, 0b11100, 1, 1, 0b011},
		{0b0011_0011_1011_0111, 0b10111, 0b11101, 0, 0, 0b011},
		{0b0011_0000_0001_0111, 0b10111, 0, 0, 0, 0b011},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		cx, cy, ntx, nty, fy := l.coarseX(), l.coarseY(), l.nametableX(), l.nametableY(), l.fineY()
		if cx != tc.wantCoarseX || cy != tc.wantCoarseY || ntx != tc.wantNameTableX || nty != tc.wantNameTableY || fy != tc.wantFineY {
			t.Errorf("%d: Got %016b, %016b, %016b, %016b, %016b, wanted %016b, %016b, %016b, %016b, %016b", i, cx, cy, ntx, nty, fy, tc.wantCoarseX, tc.wantCoarseY, tc.wantNameTableX, tc.wantNameTableY, tc.wantFineY)
		}
	}
}

func TestLoopySetCoarseX(t *testing.T) {
	cases := []struct {
		data     uint16
		ocx, ncx uint16
	}{
		{0b0000_0000_0000_0000, 0, 0},
		{0b0111_1011_1001_1000, 0b11000, 0b11100},
		{0b0011_0111_1001_0111, 0b10111, 0b11100},
		{0b0011_1111_1001_0111, 0b10111, 0b10000},
		{0b0011_0011_1011_0111, 0b10111, 0b11101},
		{0b0011_0000_0001_0111, 0b10111, 0b00100},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ocx := l.coarseX()
		l.setCoarseX(tc.ncx)
		if got := l.coarseX(); ocx != tc.ocx || got != tc.ncx {
			t.Errorf("%d: Got ocx = %05b, ncx = %05b, wanted %05b, %05b", i, ocx, got, tc.ocx, tc.ncx)

		}
	}
}

func TestLoopyIncrementCoarseX(t *testing.T) {
	cases := []struct {
		data     uint16
		ocx, ncx uint16
	}{
		{0b0000_0000_0000_0000, 0, 1},
		{0b0111_1011_1001_1000, 0b11000, 0b11001},
		{0b0011_0111_1011_0111, 0b10111, 0b11000},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ocx := l.coarseX()
		l.incrementCoarseX()
		if got := l.coarseX(); ocx != tc.ocx || got != tc.ncx {
			t.Errorf("%d: Got ocx = %05b, ncx = %05b, wanted %05b, %05b", i, ocx, got, tc.ocx, tc.ncx)

		}
	}
}

func TestLoopySetCoarseY(t *testing.T) {
	cases := []struct {
		data     uint16
		ocy, ncy uint16
	}{
		{0b0000_0000_0000_0000, 0, 0},
		{0b0111_1011_1001_1000, 0b11100, 0b11100},
		{0b0011_0111_1011_0111, 0b11101, 0b10000},
		{0b0011_1111_1111_0111, 0b11111, 0b00000},
		{0b0011_0001_0101_0111, 0b01010, 0b10101},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ocy := l.coarseY()
		l.setCoarseY(tc.ncy)
		if got := l.coarseY(); ocy != tc.ocy || got != tc.ncy {
			t.Errorf("%d: Got ocy = %05b, ncy = %05b, wanted %05b, %05b", i, ocy, got, tc.ocy, tc.ncy)

		}
	}
}

func TestLoopyIncrementCoarseY(t *testing.T) {
	cases := []struct {
		data     uint16
		ocy, ncy uint16
	}{
		{0b0000_0000_0000_0000, 0, 1},
		{0b0111_1011_1001_1000, 0b11100, 0b11101},
		{0b0011_0111_1011_0111, 0b11101, 0b11110},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ocy := l.coarseY()
		l.incrementCoarseY()
		if got := l.coarseY(); ocy != tc.ocy || got != tc.ncy {
			t.Errorf("%d: Got ocy = %05b, ncy = %05b, wanted %05b, %05b", i, ocy, got, tc.ocy, tc.ncy)

		}
	}
}

func TestLoopyToggleNametableX(t *testing.T) {
	cases := []struct {
		data     uint16
		ox, nx   uint16
		wantData uint16
	}{
		{0b0000_0000_0000_0000, 0, 1, 0b0000_0100_0000_0000},
		{0b0000_0100_0000_0000, 1, 0, 0b0000_0000_0000_0000},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ox := l.nametableX()
		l.toggleNametableX()
		if got := l.nametableX(); ox != tc.ox || got != tc.nx || l.data != tc.wantData {
			t.Errorf("%d: Got ox = %01b, nx = %01b (%016b), wanted %01b, %01b (%016b)", i, ox, got, l.data, tc.ox, tc.nx, tc.wantData)

		}
	}
}

func TestLoopyToggleNametableY(t *testing.T) {
	cases := []struct {
		data     uint16
		oy, ny   uint16
		wantData uint16
	}{
		{0b0000_0000_0000_0000, 0, 1, 0b0000_1000_0000_0000},
		{0b0000_1000_0000_0000, 1, 0, 0b0000_0000_0000_0000},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		oy := l.nametableY()
		l.toggleNametableY()
		if got := l.nametableY(); oy != tc.oy || got != tc.ny || l.data != tc.wantData {
			t.Errorf("%d: Got oy = %01b, ny = %01b (%016b), wanted %01b, %01b (%016b)", i, oy, got, l.data, tc.oy, tc.ny, tc.wantData)

		}
	}
}

func TestLoopySetFineY(t *testing.T) {
	cases := []struct {
		data     uint16
		ofy, nfy uint16
	}{
		{0b0000_0000_0000_0000, 0, 0},
		{0b0111_1011_1001_1000, 0b111, 0b101},
		{0b0011_0111_1011_0111, 0b011, 0},
		{0b0111_1111_1111_0111, 0b111, 0b010},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ofy := l.fineY()
		l.setFineY(tc.nfy)
		if got := l.fineY(); ofy != tc.ofy || got != tc.nfy {
			t.Errorf("%d: Got ofy = %03b, nfy = %03b, wanted %03b, %03b", i, ofy, got, tc.ofy, tc.nfy)

		}
	}
}

func TestLoopyIncrementFineY(t *testing.T) {
	cases := []struct {
		data     uint16
		ofy, nfy uint16
	}{
		{0b0000_0000_0000_0000, 0, 1},
		{0b0110_1011_1001_1000, 0b110, 0b111},
		{0b0011_0111_1011_0111, 0b011, 0b100},
	}

	for i, tc := range cases {
		l := &loopy{tc.data}

		ofy := l.fineY()
		l.incrementFineY()
		if got := l.fineY(); ofy != tc.ofy || got != tc.nfy {
			t.Errorf("%d: Got ofy = %03b, nfy = %03b, wanted %03b, %03b", i, ofy, got, tc.ofy, tc.nfy)

		}
	}
}

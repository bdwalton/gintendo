package nesrom

import (
	"reflect"
	"testing"
)

func TestParseHeader(t *testing.T) {
	cases := []struct {
		bytes      []byte
		wantHeader *header
	}{
		{
			[]byte{0x4e, 0x45, 0x53, 0x1a, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, &header{constant: "NES\x1a", prgSize: 2, chrSize: 1, flags6: 1, flags7: 0, flags8: 0, flags9: 0, flags10: 0, flags11: 0, flags12: 0, flags13: 0, flags14: 0, flags15: 0},
		},
	}
	for i, tc := range cases {

		if h := parseHeader(tc.bytes); !reflect.DeepEqual(h, tc.wantHeader) {
			t.Errorf("%d: Got %q, wanted %q", i, h, tc.wantHeader)
		}
	}
}

func TestNES2Format(t *testing.T) {
	h := &header{}
	cases := []struct {
		constant           string
		flags7             uint8
		wantINES, wantNES2 bool
	}{
		{"NES\x1A", 0x08, true, true},
		{"NES\x1A", 0x0C, true, false},
		{"BOB\x1A", 0x10, false, false},
		{"BOB\x1A", 0x04, false, false},
		{"BOB\x1A", 0x08, false, false},
	}

	for i, tc := range cases {
		h.constant = tc.constant
		h.flags7 = tc.flags7
		if h.isINesFormat() != tc.wantINES || h.isNES2Format() != tc.wantNES2 {
			t.Errorf("%d: ines = %t want %t; nes2 = %t, want %t", i, h.isINesFormat(), tc.wantINES, h.isNES2Format(), tc.wantNES2)
		}
	}
}

func TestMapperNum(t *testing.T) {
	h := &header{constant: "NES\x1A"}
	cases := []struct {
		flags6, flags7, flags11, flags12, flags13, flags14, flags15 uint8 // where the mapper num is assembled from
		want                                                        uint8
	}{
		{0xEF, 0xF0, 0, 0, 0, 0, 0, 0xFE}, // Not NES2, last 4 bytes 0
		{0xFF, 0xE0, 0, 0, 0, 0, 0, 0xEF}, // Not NES2, last 4 bytes 0
		{0xC0, 0xB0, 0, 0, 1, 1, 1, 0x0C}, // Not NES2, last 4 bytes not 0
		{0x1F, 0x20, 0, 0, 1, 1, 1, 0x01}, // Not NES2, last 4 bytes not 0
		{0xFF, 0xF8, 0, 0, 0, 1, 1, 0xFF}, // NES2, last 4 bytes not 0
		{0xAF, 0xD8, 0, 0, 0, 0, 0, 0xDA}, // NES2, last 4 bytes 0
	}

	for i, tc := range cases {
		h.flags6 = tc.flags6
		h.flags7 = tc.flags7
		h.flags11 = tc.flags11
		h.flags12 = tc.flags12
		h.flags13 = tc.flags13
		h.flags14 = tc.flags14
		h.flags15 = tc.flags15
		if got := h.mapperNum(); got != tc.want {
			t.Errorf("%d: Got %d, want %d", i, got, tc.want)
		}
	}
}

func TestHasTrainer(t *testing.T) {
	h := &header{constant: "NES\x1A"}
	cases := []struct {
		flags6 uint8 // where the trainer bit is stored
		want   bool
	}{
		{0xFF, true},
		{0x04, true},
		{0x0C, true},
		{0x0A, false},
	}

	for i, tc := range cases {
		h.flags6 = tc.flags6
		if got := h.hasTrainer(); got != tc.want {
			t.Errorf("%d: Got %t, want %t", i, got, tc.want)
		}
	}
}

func TestHasPlayChoice10(t *testing.T) {
	h := &header{constant: "NES\x1A"}
	cases := []struct {
		flags7 uint8 // where the playchoice10 bit is stored
		want   bool
	}{
		{0xFF, true},
		{0x02, true},
		{0x0D, false},
		{0x01, false},
	}

	for i, tc := range cases {
		h.flags7 = tc.flags7
		if got := h.hasPlayChoice(); got != tc.want {
			t.Errorf("%d: Got %t, want %t", i, got, tc.want)
		}
	}
}

func TestMirroringMode(t *testing.T) {
	h := &header{constant: "NES\x1A"}
	cases := []struct {
		flags6 uint8
		want   uint8
	}{
		{0xFF, MIRROR_FOUR_SCREEN},
		{0x00, MIRROR_HORIZONTAL},
		{0x01, MIRROR_VERTICAL},
		{0x08, MIRROR_FOUR_SCREEN},
		{0x09, MIRROR_FOUR_SCREEN},
	}

	for i, tc := range cases {
		h.flags6 = tc.flags6
		if got := h.mirroringMode(); got != tc.want {
			t.Errorf("%d: Got %d, want %d.", i, got, tc.want)
		}
	}
}

func TestBatteryBackedSRAM(t *testing.T) {
	h := &header{constant: "NES\x1A"}
	cases := []struct {
		flags6, flags8 uint8
		want           bool
		wantSize       uint8
	}{
		{0, 0, false, 0},
		{0, 16, false, 0},
		{BATTERY_BACKED_SRAM, 0, true, 1},
		{BATTERY_BACKED_SRAM, 1, true, 1},
		{BATTERY_BACKED_SRAM, 16, true, 16},
	}

	for i, tc := range cases {
		h.flags6 = tc.flags6
		h.flags8 = tc.flags8
		if got, size := h.hasPrgRAM(), h.prgRAMSize(); got != tc.want || size != tc.wantSize {
			t.Errorf("%d: Got %t, wanted %t, size = %d, wanted %d", i, got, tc.want, size, tc.wantSize)
		}
	}
}

package nesFormat

import (
	"reflect"
	"testing"
)

func TestParseHeader(t *testing.T) {
	cases := []struct {
		bytes      []byte
		wantHeader *Header
	}{
		{
			[]byte{0x4e, 0x45, 0x53, 0x1a, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, &Header{constant: "NES\x1a", prgSize: 2, chrSize: 1, flags6: 1, flags7: 0, flags8: 0, flags9: 0, flags10: 0, unused: ""},
		},
	}
	for i, tc := range cases {

		if h := parseHeader(tc.bytes); !reflect.DeepEqual(h, tc.wantHeader) {
			t.Errorf("%d: Got %q, wanted %q", i, h, tc.wantHeader)
		}
	}
}

func TestNES2Format(t *testing.T) {
	h := &Header{}
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
		if h.IsINesFormat() != tc.wantINES || h.IsNES2Format() != tc.wantNES2 {
			t.Errorf("%d: ines = %t want %t; nes2 = %t, want %t", i, h.IsINesFormat(), tc.wantINES, h.IsNES2Format(), tc.wantNES2)
		}
	}
}

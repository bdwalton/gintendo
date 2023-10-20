package ines

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

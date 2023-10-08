package mos6502

import (
	"errors"
	"testing"
)

func TestGetInst(t *testing.T) {
	cpu := New()
	cases := []struct {
		val     uint8
		want    opcode
		wantErr error
	}{
		{0x00, opcode{BRK, IMPLICIT}, nil},
		{0x24, opcode{BIT, ZERO_PAGE}, nil},
		{0x02, opcode{BRK, IMPLICIT}, invalidInstruction},
	}

	for i, tc := range cases {
		cpu.memory[0] = tc.val
		got, err := cpu.getInst()
		if got != tc.want || (err != nil && tc.wantErr == nil) || !errors.Is(err, tc.wantErr) {
			t.Errorf("%d: got %s, want %s; err %v, wantErr %v", i, got, tc.want, err, tc.wantErr)
		}
	}

}

func TestOpSEC(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x00, 0x01},
		{0xF0, 0xF1},
		{0xFF, 0xFF},
		{0xFE, 0xFF},
		{0xFF, 0xFF}, // Make sure it doesn't unset
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opSEC(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpSED(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x00, 0x08},
		{0xF0, 0xF8},
		{0xFF, 0xFF},
		{0xF9, 0xF9},
		{0xFF, 0xFF}, // Make sure it doesn't unset
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opSED(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpCLC(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x01, 0x00},
		{0xF1, 0xF0},
		{0xFF, 0xFE},
		{0xF0, 0xF0}, // Make sure it never sets instead
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opCLC(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpCLD(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x08, 0x00},
		{0xF8, 0xF0},
		{0xFF, 0xF7},
		{0xF0, 0xF0}, // Make sure it never sets instead
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opCLD(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

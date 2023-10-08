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
		{0xFE, 0xFF},
		{0xFF, 0xFF},
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
		{0xF9, 0xF9},
		{0xFF, 0xFF},
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opSED(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpSEI(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x00, 0x04},
		{0xF0, 0xF4},
		{0xFF, 0xFF},
		{0xF3, 0xF7},
		{0xFF, 0xFF},
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opSEI(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted 0x%02x, got 0x%02x", i, tc.want, cpu.status)
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
		{0xF0, 0xF0},
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
		{0xF0, 0xF0},
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opCLD(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpCLI(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x04, 0x00},
		{0xF4, 0xF0},
		{0xFF, 0xFB},
		{0xF0, 0xF0},
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opCLI(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpCLV(t *testing.T) {
	cpu := New()
	cases := []struct {
		status uint8
		want   uint8
	}{
		{0x40, 0x00},
		{0x4F, 0x0F},
		{0xFF, 0xBF},
		{0x0F, 0x0F},
	}

	for i, tc := range cases {
		cpu.status = tc.status
		cpu.opCLV(IMPLICIT)
		if cpu.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, cpu.status)
		}
	}
}

func TestOpDEX(t *testing.T) {
	cpu := New()
	cases := []struct {
		x          uint8
		status     uint8
		wantX      uint8
		wantStatus uint8
	}{
		{1, 0x00, 0, 0x02},
		{0, 0x00, 255, 0x80},
		{128, 0x00, 127, 0x00},
		{255, 0x00, 254, 0x80},
	}

	for i, tc := range cases {
		cpu.x = tc.x
		cpu.status = tc.status
		cpu.opDEX(IMPLICIT)
		if cpu.x != tc.wantX || cpu.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantX, tc.wantStatus, cpu.x, cpu.status)
		}
	}
}

func TestOpINX(t *testing.T) {
	cpu := New()
	cases := []struct {
		x          uint8
		status     uint8
		wantX      uint8
		wantStatus uint8
	}{
		{1, 0x00, 2, 0x00},
		{126, 0x00, 127, 0x00},
		{127, 0x00, 128, 0x80},
		{255, 0x00, 0, 0x02},
	}

	for i, tc := range cases {
		cpu.x = tc.x
		cpu.status = tc.status
		cpu.opINX(IMPLICIT)
		if cpu.x != tc.wantX || cpu.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantX, tc.wantStatus, cpu.x, cpu.status)
		}
	}
}

func TestOpDEY(t *testing.T) {
	cpu := New()
	cases := []struct {
		y          uint8
		status     uint8
		wantY      uint8
		wantStatus uint8
	}{
		{1, 0x00, 0, 0x02},
		{0, 0x00, 255, 0x80},
		{255, 0x00, 254, 0x80},
		{128, 0x00, 127, 0x00},
	}

	for i, tc := range cases {
		cpu.y = tc.y
		cpu.status = tc.status
		cpu.opDEY(IMPLICIT)
		if cpu.y != tc.wantY || cpu.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantY, tc.wantStatus, cpu.y, cpu.status)
		}
	}
}

func TestOpINY(t *testing.T) {
	cpu := New()
	cases := []struct {
		y          uint8
		status     uint8
		wantY      uint8
		wantStatus uint8
	}{
		{1, 0x00, 2, 0x00},
		{255, 0x00, 0, 0x02},
		{127, 0x00, 128, 0x80},
		{254, 0x00, 255, 0x80},
	}

	for i, tc := range cases {
		cpu.y = tc.y
		cpu.status = tc.status
		cpu.opINY(IMPLICIT)
		if cpu.y != tc.wantY || cpu.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantY, tc.wantStatus, cpu.y, cpu.status)
		}
	}
}

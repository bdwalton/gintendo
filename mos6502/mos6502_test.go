package mos6502

import (
	"errors"
	"testing"
)

func TestMemRead(t *testing.T) {
	cpu := New()
	cases := []struct {
		mem1 uint8
		want uint16
	}{
		{0xFF, 0xFF},
		{0x11, 0x11},
	}

	for i, tc := range cases {
		cpu.memory[i] = tc.mem1
		cpu.pc = uint16(i)
		if got := cpu.memRead16(cpu.pc); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestMemWrite(t *testing.T) {
	cpu := New()
	cases := []struct {
		mem1 uint8
		want uint8
	}{
		{0xFF, 0xFF},
		{0x11, 0x11},
	}

	for i, tc := range cases {
		cpu.pc = uint16(i)
		cpu.writeMem(cpu.pc, tc.mem1)
		if got := cpu.memRead(cpu.pc); got != tc.want {
			t.Errorf("%d: Got 0x%02x, want 0x%02x", i, got, tc.want)
		}
	}
}

func TestMemRead16(t *testing.T) {
	cpu := New()
	cases := []struct {
		mem1, mem2 uint8
		want       uint16
	}{
		{0xFF, 0x11, 0x11FF},
		{0xFF, 0x11, 0x11FF},
	}

	for i, tc := range cases {
		cpu.memory[i] = tc.mem1
		cpu.memory[i+1] = tc.mem2
		cpu.pc = uint16(i)
		if got := cpu.memRead16(cpu.pc); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestMemWrite16(t *testing.T) {
	cpu := New()
	cases := []struct {
		val        uint16
		mem1, mem2 uint8
	}{
		{0x11FF, 0xFF, 0x11},
		{0x5566, 0x66, 0x55},
	}

	for i, tc := range cases {
		cpu.pc = uint16(i)
		cpu.writeMem16(cpu.pc, tc.val)
		cpu.memory[i] = tc.mem1
		cpu.memory[i+1] = tc.mem2

		if cpu.memory[i] != tc.mem1 || cpu.memory[i+1] != tc.mem2 {
			t.Errorf("%d: Got (0x%02x, 0x%02x), want (0x%02x, 0x%02x)", i, cpu.memory[i], cpu.memory[i+1], tc.mem1, tc.mem2)
		}
	}
}

func TestGetOperandAddr(t *testing.T) {
	cpu := New()
	cpu.pc = 0x64
	cpu.memory[0x0F] = 0x44
	cpu.memory[0x10] = 0x55
	cpu.memory[cpu.pc] = 0x0F
	cpu.memory[cpu.pc+1] = 0x11
	cpu.memory[0x001F] = 0x55
	cpu.memory[0x110F] = 0xFA
	cpu.memory[0x1110] = 0xBB
	cpu.x = 0x10
	cpu.y = 0xAC

	cases := []struct {
		mode uint8
		want uint16
	}{
		{IMMEDIATE, 0x64},     // Should just return program counter
		{ZERO_PAGE, 0x000F},   // mem[pc]
		{ZERO_PAGE_X, 0x001F}, // mem[pc] + x
		{ZERO_PAGE_Y, 0x00BB}, // mem[pc] + y
		{RELATIVE, 0x73},      // pc + int8(mem[pc])
		{ABSOLUTE, 0x110F},    // mem[pc+1] << 8 + mem[pc]
		{ABSOLUTE_X, 0x111F},  // (mem[pc+1] << 8 + mem[pc]) + x
		{ABSOLUTE_Y, 0x11BB},  // (mem[pc+1] << 8 + mem[pc]) + y
		{INDIRECT, 0xBBFA},    // a = (mem[pc+1] << 8 + mem[pc]); (mem[a+1] + mem[a])
		{INDIRECT_X, 0x0055},  // mem[mem[pc] + x] (mem[pc] + x is wrapped in uint8)
		{INDIRECT_Y, 0x55F0},  // m = mem[pc]; (mem[m+1] << 8 + mem[m]) + y
	}

	for i, tc := range cases {
		if got := cpu.getOperandAddr(tc.mode); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestGetInst(t *testing.T) {
	cpu := New()
	cases := []struct {
		val     uint8
		want    opcode
		wantErr error
	}{
		{0x00, opcode{BRK, IMPLICIT, 1, 7}, nil},
		{0x24, opcode{BIT, ZERO_PAGE, 2, 3}, nil},
		{0x02, opcode{BRK, IMPLICIT, 1, 7}, invalidInstruction},
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

func memInit(val uint8) (mem [MEM_SIZE]uint8) {
	for i := 0; i < MEM_SIZE; i++ {
		mem[i] = val
	}
	return
}
func TestOpNOP(t *testing.T) {
	cpu := New()
	cpu.memory = memInit(0xEA) // NOP

	cases := []struct {
		pc         uint16
		status     uint8
		wantPC     uint16
		wantStatus uint8
	}{
		{0, 0xFF, 1, 0xFF},
		{10, 0x00, 11, 0x00},
	}

	for i, tc := range cases {
		cpu.pc = tc.pc
		cpu.status = tc.status
		cpu.step()
		if cpu.pc != tc.wantPC || cpu.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status 0x%02x), got %d (status: 0x%02x)", i, tc.wantPC, tc.wantStatus, cpu.pc, cpu.status)
		}
	}
}

func TestOpAND(t *testing.T) {
	cpu := New()
	cases := []struct {
		acc        uint8
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x01, 0x00, 0x02},
		{0x01, 0x01, 0x01, 0x00},
		{0xFF, 0xF0, 0xF0, 0x80},
	}

	for i, tc := range cases {
		cpu.pc = 0
		cpu.status = 0
		cpu.memory[cpu.pc] = tc.op1
		cpu.acc = tc.acc

		if cpu.opAND(IMMEDIATE); cpu.acc != tc.want || cpu.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, cpu.acc, cpu.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpEOR(t *testing.T) {
	cpu := New()
	cases := []struct {
		acc        uint8
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x01, 0x01, 0x00},
		{0x01, 0x01, 0x00, 0x02},
		{0xFF, 0xF0, 0x0F, 0x00},
		{0xFF, 0x0F, 0xF0, 0x80},
	}

	for i, tc := range cases {
		cpu.pc = 0
		cpu.status = 0
		cpu.memory[cpu.pc] = tc.op1
		cpu.acc = tc.acc

		if cpu.opEOR(IMMEDIATE); cpu.acc != tc.want || cpu.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, cpu.acc, cpu.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpDEC(t *testing.T) {
	cpu := New()
	cases := []struct {
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0xFF, 0x80},
		{0x01, 0x00, 0x02},
		{0xFF, 0xFE, 0x80},
		{0x02, 0x01, 0x00},
	}

	for i, tc := range cases {
		cpu.pc = 0
		cpu.status = 0
		cpu.memory[cpu.pc] = tc.op1

		if cpu.opDEC(IMMEDIATE); cpu.memory[cpu.pc] != tc.want || cpu.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, cpu.acc, cpu.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpINC(t *testing.T) {
	cpu := New()
	cases := []struct {
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x01, 0x00},
		{0xFF, 0x00, 0x02},
		{0xFE, 0xFF, 0x80},
	}

	for i, tc := range cases {
		cpu.pc = 0
		cpu.status = 0
		cpu.memory[cpu.pc] = tc.op1

		if cpu.opINC(IMMEDIATE); cpu.memory[cpu.pc] != tc.want || cpu.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, cpu.memory[cpu.pc], cpu.status, tc.want, tc.wantStatus)
		}
	}
}

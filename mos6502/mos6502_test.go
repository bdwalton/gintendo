package mos6502

import (
	"errors"
	"os"
	"testing"
)

func memInit(c *CPU, val uint8) {
	for i := 0; i < MEM_SIZE; i++ {
		c.Write(uint16(i), val)
	}
	return
}

type mem struct {
	data []uint8
}

func (m *mem) Read(addr uint16) uint8 {
	return m.data[addr]
}

func (m *mem) Write(addr uint16, val uint8) {
	m.data[addr] = val
}

func NewMem() *mem {
	return &mem{data: make([]uint8, MEM_SIZE)}
}

var cpu *CPU = New(NewMem())

func TestCycles(t *testing.T) {
	c := cpu
	memInit(c, 0xEA)

	cases := []struct {
		pc                uint16
		status, acc, x, y uint8
		op, arg1, arg2    uint8
		wantPC            uint16
		wantCycles        int
	}{
		{0, 0, 0, 0, 0, 0x69 /* ADC IMM */, 0, 0, 0x02, 2},
		{0, 0, 0, 0, 0, 0x7D /* ADC ABS_X */, 0, 0, 0x03, 4 /* no page crossed */},
		{0xFF, 0, 1, 1, 0, 0x7D /* ADC ABS_X */, 0xFF, 0x01, 0x0102, 5 /* page crossed*/},
		{0xFF, 0, 1, 1, 2, 0x79 /* ADC ABS_Y */, 0xFF, 0x01, 0x0102, 5 /* page crossed*/},
		{0xFF, 0, 1, 1, 0, 0x79 /* ADC ABS_Y */, 0xFF, 0x01, 0x0102, 4 /* no page crossed*/},
		{0, 0 /* CARRY CLEAR */, 1, 1, 0, 0x90 /* BCC REL */, 0x20, 0x01, 0x22, 3 /* branch succeed, no page crossed*/},
		{0xFF, 0 /* CARRY CLEAR */, 1, 1, 0, 0x90 /* BCC REL */, 10, 0x01, 0x010b, 4 /* branch succeed, page crossed*/},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.acc = tc.acc
		c.x = tc.x
		c.y = tc.y
		c.Write(c.pc, tc.op)
		c.Write(c.pc+1, tc.arg1)
		c.Write(c.pc+2, tc.arg2)

		c.cycles = 0 // So we execute op

		c.Step()

		if c.cycles != tc.wantCycles || c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, cycles = %d, wanted PC = 0x%04x, cycles %d.", i, c.pc, c.cycles, tc.wantPC, tc.wantCycles)
		}
	}
}

func TestEncodeBCD(t *testing.T) {
	cases := []struct {
		decimal, bcd uint8
	}{
		{99, 0x99},
		{70, 0x70},
		{85, 0x85},
		{1, 0x01},
		{00, 0x00},
	}

	for i, tc := range cases {
		if got := encodeBCD(tc.decimal); got != tc.bcd {
			t.Errorf("%d: Got 0x%02x from %d, wanted 0x%02x", i, got, tc.decimal, tc.bcd)
		}
	}
}

func TestDecodeBCD(t *testing.T) {
	cases := []struct {
		bcd, decimal uint8
	}{
		{0x99, 99},
		{0x70, 70},
		{0x85, 85},
		{0x01, 1},
		{0x00, 00},
	}

	for i, tc := range cases {
		if got := decodeBCD(tc.bcd); got != tc.decimal {
			t.Errorf("%d: Got %d from 0x%02x, wanted %d", i, got, tc.bcd, tc.decimal)
		}
	}
}

func TestMemRead(t *testing.T) {
	c := cpu
	cases := []struct {
		mem1 uint8
		want uint8
	}{
		{0xFF, 0xFF},
		{0x11, 0x11},
	}

	for i, tc := range cases {
		c.Write(uint16(i), tc.mem1)
		c.pc = uint16(i)
		if got := c.Read(c.pc); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestMemWrite(t *testing.T) {
	c := cpu
	cases := []struct {
		mem1 uint8
		want uint8
	}{
		{0xFF, 0xFF},
		{0x11, 0x11},
	}

	for i, tc := range cases {
		c.pc = uint16(i)
		c.Write(c.pc, tc.mem1)
		if got := c.Read(c.pc); got != tc.want {
			t.Errorf("%d: Got 0x%02x, want 0x%02x", i, got, tc.want)
		}
	}
}

func TestMemRead16(t *testing.T) {
	c := cpu
	cases := []struct {
		mem1, mem2 uint8
		want       uint16
	}{
		{0xFF, 0x11, 0x11FF},
		{0xFF, 0x11, 0x11FF},
	}

	for i, tc := range cases {
		c.Write(uint16(i), tc.mem1)
		c.Write(uint16(i+1), tc.mem2)
		c.pc = uint16(i)
		if got := c.Read16(c.pc); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestMemWrite16(t *testing.T) {
	c := cpu
	cases := []struct {
		val        uint16
		mem1, mem2 uint8
	}{
		{0x11FF, 0xFF, 0x11},
		{0x5566, 0x66, 0x55},
	}

	for i, tc := range cases {
		c.pc = uint16(i)
		c.Write16(c.pc, tc.val)
		c.Write(uint16(i), tc.mem1)
		c.Write(uint16(i+1), tc.mem2)

		m1, m2 := c.Read(uint16(i)), c.Read(uint16(i+1))
		if m1 != tc.mem1 || m2 != tc.mem2 {
			t.Errorf("%d: Got (0x%02x, 0x%02x), want (0x%02x, 0x%02x)", i, m1, m2, tc.mem1, tc.mem2)
		}
	}
}

func TestPushAddress(t *testing.T) {
	c := cpu
	cases := []struct {
		addr                   uint16
		sp                     uint8
		wantLO, wantHI, wantSP uint8
	}{
		{0xF101, 0xFF, 0x01, 0xF1, 0xFD},
		{0xAC08, 0x10, 0x08, 0xAC, 0x0E},
	}

	for i, tc := range cases {
		c.sp = tc.sp
		c.pushAddress(tc.addr)
		if c.sp != tc.wantSP || c.Read(c.StackAddr()+2) != tc.wantHI || c.Read(c.StackAddr()+1) != tc.wantLO {
			top := c.StackAddr() + 2
			bottom := top - 1
			t.Errorf("%d: Got 0x%02x %v, want 0x%02x %v", i, c.sp, c.memRange(bottom, top), tc.wantSP, []uint8{tc.wantLO, tc.wantHI})
		}

	}
}

func TestPopAddress(t *testing.T) {
	c := cpu
	cases := []struct {
		addr     uint16
		sp       uint8
		wantSP   uint8
		wantAddr uint16
	}{
		{0xFF01, 0xF3, 0xF3, 0xFF01},
	}

	for i, tc := range cases {
		c.sp = tc.sp
		c.pushAddress(tc.addr)

		if addr := c.popAddress(); c.sp != tc.wantSP || addr != tc.wantAddr {

			t.Errorf("%d: Got 0x%02x (sp 0x%02x), want 0x%02x (sp 0x%02x)", i, addr, c.sp, tc.wantAddr, tc.wantSP)

		}

	}
}

func TestGetOperandAddr(t *testing.T) {
	c := cpu

	c.Write16(0x000F, 0x5544)
	c.Write16(0x0064, 0x110F)
	c.Write16(0x001F, 0x0055)
	c.Write16(0x110F, 0xBBFA)
	c.Write(0xFF66, 0x82)
	c.x = 0x10
	c.y = 0xAC

	cases := []struct {
		pc   uint16 // first operand, not op
		mode uint8
		want uint16
	}{
		{0x0064, IMMEDIATE, 0x64},     // Should just return program counter
		{0x0064, ZERO_PAGE, 0x000F},   // mem[pc]
		{0x0064, ZERO_PAGE_X, 0x001F}, // mem[pc] + x
		{0x0064, ZERO_PAGE_Y, 0x00BB}, // mem[pc] + y
		{0x0064, RELATIVE, 0x74},      // pc + int8(mem[pc])
		{0xFF66, RELATIVE, 0xFEE9},    // pc - int8(mem[pc])
		{0x0064, ABSOLUTE, 0x110F},    // mem[pc+1] << 8 + mem[pc]
		{0x0064, ABSOLUTE_X, 0x111F},  // (mem[pc+1] << 8 + mem[pc]) + x
		{0x0064, ABSOLUTE_Y, 0x11BB},  // (mem[pc+1] << 8 + mem[pc]) + y
		{0x0064, INDIRECT, 0xBBFA},    // a = (mem[pc+1] << 8 + mem[pc]); (mem[a+1] + mem[a])
		{0x0064, INDIRECT_X, 0x0055},  // mem[mem[pc] + x] (mem[pc] + x is wrapped in uint8)
		{0x0064, INDIRECT_Y, 0x55F0},  // m = mem[pc]; (mem[m+1] << 8 + mem[m]) + y
	}

	for i, tc := range cases {
		c.pc = tc.pc
		if got := c.getOperandAddr(tc.mode); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestGetInst(t *testing.T) {
	c := cpu
	cases := []struct {
		val     uint8
		want    opcode
		wantErr error
	}{
		{0x00, opcode{BRK, "BRK", IMPLICIT, 2, 7}, nil},
		{0x24, opcode{BIT, "BIT", ZERO_PAGE, 2, 3}, nil},
		{0x02, opcode{}, invalidInstruction},
	}

	for i, tc := range cases {
		c.pc = 0
		c.cycles = 0
		c.Write(0, tc.val)
		got, err := c.getInst()
		if got != tc.want || (err != nil && tc.wantErr == nil) || !errors.Is(err, tc.wantErr) {
			t.Errorf("%d: got %s, want %s; err %v, wantErr %v", i, got, tc.want, err, tc.wantErr)
		}
	}

}

func TestReset(t *testing.T) {
	c := cpu
	cases := []struct {
		int_reset_pc uint16
		wantPC       uint16
	}{
		{0x0567, 0x0567},
		{0xAC13, 0xAC13},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0
		c.Write16(INT_RESET, tc.int_reset_pc)
		c.Reset()

		if c.pc != tc.wantPC || c.status != 0x24 {
			t.Errorf("%d: PC = 0x%04x (status 0x%02x), wanted 0x%04x (status 0x%02x)", i, c.pc, c.status, tc.wantPC, 0x24)
		}
	}
}

func TestOpADC(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, op1, status uint8
		want, wantStatus uint8
	}{
		// Decimal addition
		{0xFF, 0x01, 0x00, 0x00, 0x03 /* ZERO, CARRY */},
		{0xF1, 0x01, 0x00, 0xF2, 0x80 /* NEGATIVE */},
		{0x00, 0x00, 0x00, 0x00, 0x02 /* ZERO */},
		{0xF0, 0x0F, 0x00, 0xFF, 0x80 /* NEGATIVE */},
		{0xFF, 0xF0, 0x01 /* CARRY */, 0xF0, 0x81 /* NEGATIVE, CARRY */},
		{0xEF, 0xE1, 0x00, 0xD0, 0x81 /* NEGATIVE, CARRY */},
		// BCD addition
		{0x54, 0x99, 0x09 /* DECIMAL, CARRY */, 0x54, 0x09 /* DECIMAL, CARRY */},
		{0x54, 0x99, 0x08 /* DECIMAL */, 0x53, 0x09 /* DECIMAL, CARRY */},
		{0x00, 0x99, 0x08 /* DECIMAL */, 0x99, 0x88 /* NEGATIVE, DECIMAL */},
		{0x99, 0x01, 0x08 /* DECIMAL */, 0x00, 0x0b /* DECIMAL, ZERO, CARRY */},
		{0x99, 0x00, 0x09 /* DECIMAL, CARRY */, 0x00, 0x0b /* DECIMAL, ZERO, CARRY */},
		{0x99, 0x01, 0x09 /* DECIMAL, CARRY */, 0x01, 0x09 /* DECIMAL, CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0x7780
		c.acc = tc.acc
		c.status = tc.status
		c.Write(c.pc, tc.op1)

		if c.ADC(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (status 0x%02x), wanted 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpAND(t *testing.T) {
	c := cpu
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
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)
		c.acc = tc.acc

		if c.AND(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpASL(t *testing.T) {
	c := cpu
	cases := []struct {
		val, mode        uint8 // ACCUMULATOR and ZERO_PAGE are what we use for testing
		want, wantStatus uint8
	}{
		{0x01, ACCUMULATOR, 0x02, 0x00},
		{0x81, ACCUMULATOR, 0x02, 0x01 /* CARRY */},
		{0xD1, ACCUMULATOR, 0xa2, 0x81 /* NEGATIVE, CARRY */},
		{0x01, ZERO_PAGE, 0x02, 0x00},
		{0x81, ZERO_PAGE, 0x02, 0x01 /* CARRY */},
		{0xD1, ZERO_PAGE, 0xa2, 0x81 /* NEGATIVE, CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0x000F
		c.status = 0 // Clear processor init defaults
		switch tc.mode {
		case ACCUMULATOR:
			c.acc = tc.val
		default:
			c.Write(c.getOperandAddr(tc.mode), tc.val)
		}

		c.ASL(tc.mode)

		var got uint8
		switch tc.mode {
		case ACCUMULATOR:
			got = c.acc
		default:
			got = c.Read(c.getOperandAddr(tc.mode))
		}
		if got != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, status 0x%02x; Want 0x%02x, status 0x%02x", i, got, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpBCC(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16 // operand, so 1 beyond pc for op
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x01 /* CARRY */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x01 /* CARRY */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6682},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BCC(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBCS(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x01 /* CARRY */, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x01 /* CARRY */, 0x6682},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BCS(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBEQ(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x02 /* ZERO */, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x02 /* ZERO */, 0x6682},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BEQ(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBIT(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, op    uint8
		wantStatus uint8
	}{
		{0x01, 0x01, 0x00},
		{0x81, 0x01, 0x00},
		{0x00, 0x01, 0x02 /* ZERO */},
		{0x00, 0x81, 0x82 /* NEGATIVE, ZERO */},
		{0x00, 0xC1, 0xC2 /* NEGATIVE, OVERFLOW, ZERO */},
		{0x00, 0xE1, 0xC2 /* NEGATIVE, OVERFLOW, ZERO */},
		{0x01, 0xE1, 0xC0 /* NEGATIVE, OVERFLOW */},
	}

	for i, tc := range cases {
		c.pc = 0x0300
		c.status = 0 // Clear processor init defaults
		c.acc = tc.acc
		c.Write(c.getOperandAddr(ZERO_PAGE), tc.op)

		if c.BIT(ZERO_PAGE); c.status != tc.wantStatus {
			t.Errorf("%d: Got status = 0x%02x, wanted 0x%02x", i, c.status, tc.wantStatus)
		}
	}
}

func TestOpBMI(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x80 /* NEGATIVE */, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x80 /* NEGATIVE */, 0x6682},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BMI(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBNE(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16 // first operand, not op, so branching from pc-1
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x02 /* ZERO */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x02 /* ZERO */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6682},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BNE(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBPL(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16 // first operand, not op, so branching from pc-1
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x80 /* NEGATIVE */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x80 /* NEGATIVE */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6682},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BPL(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBRK(t *testing.T) {
	c := cpu
	cases := []struct {
		pc         uint16
		brk        uint16
		status     uint8
		wantPC     uint16
		wantReturn uint16
		wantStatus uint8
		wantStStat uint8
	}{
		{0xFF15, 0xAC69, 0x00, 0xAC69, 0xFF16, 0x04 /* I set */, 0x10 /* BRK */},
		{0xAAAA, 0x1167, 0x81, 0x1167, 0xAAAB, 0x85 /* N,I,C set */, 0x91 /* N,B,C */},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write16(INT_BRK, tc.brk)
		c.BRK(IMPLICIT)
		stStat := c.popStack()
		ret := c.popAddress()
		if c.pc != tc.wantPC || c.status != tc.wantStatus || ret != tc.wantReturn || stStat != tc.wantStStat {
			t.Errorf("%d: PC = 0x%04x (status 0x%02x), wanted 0x%04x (status 0x%02x)", i, c.pc, c.status, tc.wantPC, tc.wantStatus)
		}
	}
}

func TestOpBVC(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16 // first operand, not op, so branching from pc-1
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x40 /* OVERFLOW */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x40 /* OVERFLOW */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6682},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BVC(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBVS(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16 // first operand, not op, so branching from pc-1
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x40 /* OVERFLOW */, 0x666E},
		{0x6677, 0x0A /* +10 */, 0x40 /* OVERFLOW */, 0x6682},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.Write(c.pc, tc.offset)
		c.BVS(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpCLC(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.CLC(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCLD(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.CLD(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCLI(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.CLI(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCLV(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.CLV(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCMP(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, m     uint8
		wantStatus uint8
	}{
		{0x41, 0x41, 0x03 /* ZERO, CARRY */},
		{0x41, 0x42, 0x80 /* NEGATIVE */},
		{0x10, 0x01, 0x01 /* CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0 // Clear processor init defaults
		c.acc = tc.acc
		c.Write(c.pc, tc.m)
		if c.CMP(IMMEDIATE); c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, wanted 0x%02x", i, c.status, tc.wantStatus)
		}
	}
}

func TestOpCPX(t *testing.T) {
	c := cpu
	cases := []struct {
		x, m       uint8
		wantStatus uint8
	}{
		{0x42, 0x42, 0x03 /* ZERO, CARRY */},
		{0x42, 0x43, 0x80 /* NEGATIVE */},
		{0x11, 0x02, 0x01 /* CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0 // Clear processor init defaults
		c.x = tc.x
		c.Write(c.pc, tc.m)
		if c.CPX(IMMEDIATE); c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, wanted 0x%02x", i, c.status, tc.wantStatus)
		}
	}
}

func TestOpCPY(t *testing.T) {
	c := cpu
	cases := []struct {
		y, m       uint8
		wantStatus uint8
	}{
		{0x43, 0x43, 0x03 /* ZERO, CARRY */},
		{0x43, 0x44, 0x80 /* NEGATIVE */},
		{0x12, 0x03, 0x01 /* CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0 // Clear processor init defaults
		c.y = tc.y
		c.Write(c.pc, tc.m)
		if c.CPY(IMMEDIATE); c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, wanted 0x%02x", i, c.status, tc.wantStatus)
		}
	}
}

func TestOpDEC(t *testing.T) {
	c := cpu
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
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)

		c.DEC(IMMEDIATE)
		if m := c.Read(c.pc); m != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, m, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpDEX(t *testing.T) {
	c := cpu
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
		c.x = tc.x
		c.status = tc.status
		c.DEX(IMPLICIT)
		if c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantX, tc.wantStatus, c.x, c.status)
		}
	}
}

func TestOpDEY(t *testing.T) {
	c := cpu
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
		c.y = tc.y
		c.status = tc.status
		c.DEY(IMPLICIT)
		if c.y != tc.wantY || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantY, tc.wantStatus, c.y, c.status)
		}
	}
}

func TestOpEOR(t *testing.T) {
	c := cpu
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
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)
		c.acc = tc.acc

		c.EOR(IMMEDIATE)
		if c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpINX(t *testing.T) {
	c := cpu
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
		c.x = tc.x
		c.status = tc.status
		c.INX(IMPLICIT)
		if c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantX, tc.wantStatus, c.x, c.status)
		}
	}
}

func TestOpINY(t *testing.T) {
	c := cpu
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
		c.y = tc.y
		c.status = tc.status
		c.INY(IMPLICIT)
		if c.y != tc.wantY || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantY, tc.wantStatus, c.y, c.status)
		}
	}
}

func TestOpINC(t *testing.T) {
	c := cpu
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
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)

		c.INC(IMMEDIATE)
		if m := c.Read(c.pc); m != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, m, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpJMP(t *testing.T) {
	c := cpu
	cases := []struct {
		pc              uint16
		mode            uint8
		target, target2 uint16
		wantPC          uint16
	}{
		{0x02FF, ABSOLUTE, 0x03AC, 0x00F1, 0x03AC},
		{0x03FF, ABSOLUTE, 0x03AC, 0x5566, 0x03AC},
		{0x03FF, INDIRECT, 0x03AC, 0x6671, 0x6671},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.Write16(c.pc, tc.target)
		c.Write16(c.getOperandAddr(ABSOLUTE), tc.target2)

		c.JMP(tc.mode)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, wanted 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpJSR(t *testing.T) {
	c := cpu
	cases := []struct {
		pc               uint16
		target           uint16
		sp               uint8
		wantPC, wantAddr uint16
	}{
		{0x02FF, 0xAC01, 0xFF, 0xAC01, 0x0300},
		{0x03AB, 0xDD01, 0xFE, 0xDD01, 0x03AC},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.Write16(c.pc, tc.target)
		c.sp = tc.sp

		c.JSR(ABSOLUTE)

		if addr := c.popAddress(); c.pc != tc.wantPC || addr != tc.wantAddr {
			t.Errorf("%d: Got PC = 0x%04x, Addr = 0x%04x; Want PC = 0x%04x, Addr = 0x%04x", i, c.pc, addr, tc.wantPC, tc.wantAddr)
		}
	}
}

func TestOpLDA(t *testing.T) {
	c := cpu
	cases := []struct {
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x00, 0x02},
		{0x01, 0x01, 0x00},
		{0xFF, 0xFF, 0x80},
		{0x8F, 0x8F, 0x80},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)

		if c.LDA(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpLDX(t *testing.T) {
	c := cpu
	cases := []struct {
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x00, 0x02},
		{0x01, 0x01, 0x00},
		{0xFF, 0xFF, 0x80},
		{0x8F, 0x8F, 0x80},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)

		c.LDX(IMMEDIATE)
		if c.x != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.x, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpLDY(t *testing.T) {
	c := cpu
	cases := []struct {
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x00, 0x02},
		{0x01, 0x01, 0x00},
		{0xFF, 0xFF, 0x80},
		{0x8F, 0x8F, 0x80},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)

		c.LDY(IMMEDIATE)
		if c.y != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.y, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpLSR(t *testing.T) {
	c := cpu
	cases := []struct {
		val, mode        uint8 // ACCUMULATOR and ZERO_PAGE are what we use for testing
		want, wantStatus uint8
	}{
		{0x01, ACCUMULATOR, 0x00, 0x03 /* ZERO, CARRY */},
		{0x02, ACCUMULATOR, 0x01, 0x00},
		{0xF1, ACCUMULATOR, 0x78, 0x01 /* CARRY */},
		{0x01, ZERO_PAGE, 0x00, 0x03 /* ZERO, CARRY */},
		{0x02, ZERO_PAGE, 0x01, 0x00},
		{0xF1, ZERO_PAGE, 0x78, 0x01 /* CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0x000F
		c.status = 0 // Clear processor init defaults
		switch tc.mode {
		case ACCUMULATOR:
			c.acc = tc.val
		default:
			c.Write(c.getOperandAddr(tc.mode), tc.val)
		}

		c.LSR(tc.mode)

		var got uint8
		switch tc.mode {
		case ACCUMULATOR:
			got = c.acc
		default:
			got = c.Read(c.getOperandAddr(tc.mode))
		}
		if got != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, status 0x%02x; Want 0x%02x, status 0x%02x", i, got, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpNOP(t *testing.T) {
	c := cpu
	memInit(c, 0xEA) // NOP

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
		c.cycles = 0
		c.pc = tc.pc
		c.status = tc.status
		c.Step()
		if c.pc != tc.wantPC || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status 0x%02x), got %d (status: 0x%02x)", i, tc.wantPC, tc.wantStatus, c.pc, c.status)
		}
	}
}

func TestPCWithStep(t *testing.T) {
	c := cpu
	memInit(c, 0xEA)

	cases := []struct {
		status uint8
		inst   uint8
		m1, m2 uint8
		wantPC uint16
	}{
		{0x00 /* CARRY CLEAR */, 0x90 /* BCC */, 0xCC, 0x00, 0xFFCE},
		{0x01 /* CARRY */, 0x90 /* BCC */, 0xCC, 0x00, 0x0002},
		{0x01 /* CARRY */, 0xB0 /* BCS */, 0xCC, 0x00, 0xFFCE},
		{0x00 /* CARRY CLEAR */, 0xB0 /* BCS */, 0xCC, 0x00, 0x0002},
		{0x00 /* ZERO CLEAR */, 0xF0 /* BEQ */, 0xCC, 0x00, 0x0002},
		{0x02 /* ZERO */, 0xF0 /* BEQ */, 0x1C, 0x00, 0x001E},
		{0x00 /* NEGATIVE CLEAR */, 0x30 /* BMI */, 0x1C, 0x00, 0x0002},
		{0x80 /* NEGATIVE */, 0x30 /* BMI */, 0x1C, 0x00, 0x001E},
		{0x00 /* NEGATIVE CLEAR */, 0x10 /* BPL */, 0x1C, 0x00, 0x001E},
		{0x80 /* NEGATIVE */, 0x10 /* BPL */, 0x1C, 0x00, 0x0002},
		{0x00 /* OVERFLOW CLEAR */, 0x50 /* BVC */, 0x1C, 0x00, 0x001E},
		{0x40 /* OVERFLOW */, 0x50 /* BVC */, 0x1C, 0x00, 0x0002},
		{0x00 /* OVERFLOW CLEAR */, 0x70 /* BVS */, 0x1C, 0x00, 0x0002},
		{0x40 /* OVERFLOW */, 0x70 /* BVS */, 0x1C, 0x00, 0x001E},
		{0x00 /* EMPTY */, 0x4C /* JMP(abs) */, 0x1C, 0x1E, 0x1E1C},
		{0x00 /* EMPTY */, 0x2d /* AND(abs) */, 0x1C, 0x1E, 0x0003}, // 3 bytes
		{0x00 /* EMPTY */, 0x29 /* AND(imm) */, 0xC1, 0xE1, 0x0002}, // 2 bytes
		{0x00 /* EMPTY */, 0x18 /* CLC */, 0xC1, 0xE1, 0x0001},      // 1 byte
	}

	for i, tc := range cases {
		c.cycles = 0
		c.pc = 0 // first operand, not op, so branching from pc-1
		c.status = tc.status
		c.Write(c.pc, tc.inst)
		c.Write(c.pc+1, tc.m1)
		c.Write(c.pc+2, tc.m2)

		c.Step()
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, wanted 0x%04x.", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpORA(t *testing.T) {
	c := cpu
	cases := []struct {
		acc        uint8
		op1        uint8
		want       uint8
		wantStatus uint8
	}{
		{0x00, 0x01, 0x01, 0x00},
		{0x01, 0x01, 0x01, 0x00},
		{0x01, 0x00, 0x01, 0x00},
		{0x00, 0x00, 0x00, 0x02},
		{0xFF, 0xFF, 0xFF, 0x80},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0
		c.Write(c.pc, tc.op1)
		c.acc = tc.acc

		if c.ORA(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpPHA(t *testing.T) {
	c := cpu
	cases := []struct {
		acc    uint8
		wantSP uint8
	}{
		// These cases build on each other
		{0x01, 0xFE},
		{0x02, 0xFD},
		{0xFF, 0xFC},
	}

	// Set the stack to the top (which differs from poweron/reset value)
	c.sp = 0xFF

	for i, tc := range cases {
		c.acc = tc.acc
		c.PHA(IMPLICIT)
		if m := c.Read(c.StackAddr() + 1); m != tc.acc || c.sp != tc.wantSP {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; Mem = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, m, tc.acc)
		}
	}
}

func TestOpPHP(t *testing.T) {
	c := cpu
	cases := []struct {
		status uint8
		wantSP uint8
	}{
		// These cases build on each other
		{0x01, 0xFE},
		{0x02, 0xFD},
		{0x80, 0xFC},
	}

	// Set the stack to the top (which differs from poweron/reset value)
	c.sp = 0xFF

	for i, tc := range cases {
		c.status = tc.status
		c.PHP(IMPLICIT)
		if m := c.Read(c.StackAddr() + 1); m != (tc.status|STATUS_FLAG_BREAK) || c.sp != tc.wantSP {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; Mem = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, m, tc.status)
		}
	}
}

func TestOpPLA(t *testing.T) {
	c := cpu
	cases := []struct {
		acc        uint8
		wantSP     uint8
		wantStatus uint8
	}{
		// These cases build on each other
		{0xFE, 0xFC, 0x80},
		{0x82, 0xFD, 0x80},
		{0x00, 0xFE, 0x02},
		{0x01, 0xFF, 0x00},
	}

	// Set the stack to the top (which differs from poweron/reset value)
	c.sp = 0xFF

	// Adjust c.sp with these calls, in reverse from the cases
	// we'll compare as we pop.
	for i := len(cases); i > 0; i -= 1 {
		c.acc = cases[i-1].acc
		c.PHA(IMPLICIT)
	}

	for i, tc := range cases {
		c.acc = 0
		c.status = 0
		if c.PLA(IMPLICIT); c.sp != tc.wantSP || c.acc != tc.acc || c.status != tc.wantStatus {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; ACC = 0x%02x, want 0x%02x; Status = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, c.acc, tc.acc, c.status, tc.wantStatus)
		}
	}
}

func TestOpPLP(t *testing.T) {
	c := cpu
	cases := []struct {
		status     uint8
		wantSP     uint8
		wantStatus uint8
	}{
		// These cases build on each other
		{0x80, 0xFC, 0xa0}, /* Unused flag always on */
		{0x81, 0xFD, 0xa1},
		{0x00, 0xFE, 0x20},
		{0x01, 0xFF, 0x21},
	}

	// Set the stack to the top (which differs from poweron/reset value)
	c.sp = 0xFF

	// Adjust c.sp with these calls, in reverse from the cases
	// we'll compare as we pop.
	for i := len(cases); i > 0; i -= 1 {
		c.status = cases[i-1].status
		c.PHP(IMPLICIT) // We test that this forces B to be set
	}

	for i, tc := range cases {
		c.status = 0
		if c.PLP(IMPLICIT); c.sp != tc.wantSP || c.status != tc.wantStatus {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; Status = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, c.status, tc.wantStatus)
		}
	}
}

func TestOpROL(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, op1   uint8 // Seeded acc and memory location 0
		mode       uint8 // Addressing mode (ACCUMULATOR or ZERO_PAGE)
		status     uint8 // Current status
		want       uint8 // Value of ACC or OP1 after ROL
		wantStatus uint8 // Value of status after ROL
	}{
		{0x00, 0x00, ACCUMULATOR, 0x00, 0x00, 0x02 /* ZERO */},
		{0x01, 0x00, ACCUMULATOR, 0x00, 0x02, 0x00},
		{0x00, 0x00, ACCUMULATOR, 0x01 /* CARRY */, 0x01, 0x00},
		{0x01, 0x01, ACCUMULATOR, 0x01 /* CARRY */, 0x03, 0x00},
		{0x01, 0x01, ACCUMULATOR, 0x00, 0x02, 0x00},
		{0x80, 0x01, ACCUMULATOR, 0x00, 0x00, 0x03 /* ZERO, CARRY */},
		{0x81, 0x01, ACCUMULATOR, 0x00, 0x02, 0x01 /* CARRY */},
		{0xC1, 0x01, ACCUMULATOR, 0x00, 0x82, 0x81 /* CARRY, NEGATIVE */},
		{0x00, 0x01, ZERO_PAGE, 0x00, 0x02, 0x00},
		{0x00, 0x01, ZERO_PAGE, 0x01 /* CARRY */, 0x03, 0x00},
		{0x01, 0x01, ZERO_PAGE, 0x01 /* CARRY */, 0x03, 0x00},
		{0x01, 0x01, ZERO_PAGE, 0x00, 0x02, 0x00},
		{0x01, 0x80, ZERO_PAGE, 0x00, 0x00, 0x03 /* ZERO, CARRY */},
		{0x01, 0x81, ZERO_PAGE, 0x00, 0x02, 0x01 /* CARRY */},
		{0x01, 0xC1, ZERO_PAGE, 0x00, 0x82, 0x81 /* CARRY, NEGATIVE */},
	}

	for i, tc := range cases {
		c.pc = 0x10 // memory addr 0x10 should always be 0 on init
		c.acc = tc.acc
		if tc.mode != ACCUMULATOR {
			c.Write(c.getOperandAddr(tc.mode), tc.op1)
		}

		c.status = tc.status

		c.ROL(tc.mode)
		v := c.acc
		if tc.mode == ZERO_PAGE {
			v = c.Read(c.getOperandAddr(tc.mode)) // We don't run step(), so PC isn't updated
		}

		if v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status = 0x%02x), want 0x%02x (status = 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpROR(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, op1   uint8 // Seeded acc and memory location 0
		mode       uint8 // Addressing mode (ACCUMULATOR or ZERO_PAGE)
		status     uint8 // Current status
		want       uint8 // Value of ACC or OP1 after ROR
		wantStatus uint8 // Value of status after ROR
	}{
		{0x00, 0x00, ACCUMULATOR, 0x00, 0x00, 0x02 /* ZERO */},
		{0x00, 0x00, ACCUMULATOR, 0x01 /* CARRY */, 0x80, 0x80 /* NEGATIVE */},
		{0x40, 0x00, ACCUMULATOR, 0x01 /* CARRY */, 0xa0, 0x80 /* NEGATIVE */},
		{0x01, 0x01, ACCUMULATOR, 0x01 /* CARRY */, 0x80, 0x81 /* NEGATIVE, CARRY */},
		{0x01, 0x01, ACCUMULATOR, 0x00, 0x00, 0x03 /* ZERO, CARRY */},
		{0x80, 0x01, ACCUMULATOR, 0x00, 0x40, 0x00},
		{0x81, 0x01, ACCUMULATOR, 0x00, 0x40, 0x01 /* CARRY */},
		{0xC1, 0x01, ACCUMULATOR, 0x00, 0x60, 0x01 /* CARRY */},
		{0x00, 0x00, ZERO_PAGE, 0x00, 0x00, 0x02 /* ZERO */},
		{0x00, 0x01, ZERO_PAGE, 0x00, 0x00, 0x03 /* ZERO, CARRY */},
		{0x00, 0x02, ZERO_PAGE, 0x01, 0x81, 0x80 /* NEGATIVE */},
		{0x00, 0x01, ZERO_PAGE, 0x01 /* CARRY */, 0x80, 0x81},
		{0x00, 0x81, ZERO_PAGE, 0x00, 0x40, 0x01 /* CARRY */},
		{0x00, 0x82, ZERO_PAGE, 0x01, 0xC1, 0x80 /* NEGATIVE */},
	}

	for i, tc := range cases {
		c.pc = 0x10 // memory addr 0x10 should always be 0 on init
		c.acc = tc.acc
		if tc.mode != ACCUMULATOR {
			c.Write(c.getOperandAddr(tc.mode), tc.op1)
		}
		c.status = tc.status

		c.ROR(tc.mode)
		v := c.acc
		if tc.mode == ZERO_PAGE {
			v = c.Read(c.getOperandAddr(tc.mode)) // We don't run step(), so PC isn't updated
		}

		if v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status = 0x%02x), want 0x%02x (status = 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpRTI(t *testing.T) {
	c := cpu
	cases := []struct {
		stack      []uint8 // pc and status as 3 uint8 values
		wantPC     uint16
		wantStatus uint8
	}{
		{[]uint8{0xFF, 0x15, 0x81}, 0xFF15, 0x81},
		{[]uint8{0xAC, 0x77, 0x02}, 0xAC77, 0x02},
	}

	for i, tc := range cases {
		c.pc = 0
		c.status = 0
		for _, x := range tc.stack {
			c.pushStack(x)
		}

		c.RTI(IMPLICIT)
		if c.pc != tc.wantPC || c.status != tc.wantStatus {
			t.Errorf("%d: PC = 0x%04x (status 0x%02x), wanted 0x%04x (status 0x%02x)", i, c.pc, c.status, tc.wantPC, tc.wantStatus)

		}
	}
}

func TestOpRTS(t *testing.T) {
	c := cpu
	cases := []struct {
		pc     uint16
		target uint16
		sp     uint8
		wantPC uint16
		wantSP uint8
	}{
		{0x02AA, 0x30F1, 0xFE, 0x30F2, 0xFE},
		{0x03CA, 0x4155, 0xFF, 0x4156, 0xFF},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.sp = tc.sp
		c.pushAddress(tc.target)

		if c.RTS(IMPLICIT); c.pc != tc.wantPC || c.sp != tc.wantSP {
			t.Errorf("%d: Got PC = 0x%04x, SP = 0x%02x, want PC = 0x%04x, SP = 0x%02x", i, c.pc, c.sp, tc.wantPC, tc.wantSP)
		}
	}
}

func TestOpSBC(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, op1, status uint8
		want, wantStatus uint8
	}{
		// Decimal subtraction
		{0xFF, 0x01, 0x01, 0xFE, 0x81},
		{0x42, 0x01, 0x01, 0x41, 0x01},
		{0x42, 0x42, 0x01, 0x00, 0x03 /* ZERO, CARRY */},
		{0xD0, 0x70, 0x01, 0x60, 0x41 /* OVERFLOW, CARRY */},
		// BCD subtraction
		{0x54, 0x99, 0x09 /* DECIMAL, CARRY */, 0x55, 0x08 /* DECIMAL */},
		{0x54, 0x99, 0x08 /* DECIMAL */, 0x54, 0x08 /* DECIMAL */},
		{0x00, 0x99, 0x08 /* DECIMAL */, 0x00, 0x0a /* DECIMAL, ZERO */},
		{0x99, 0x01, 0x08 /* DECIMAL */, 0x97, 0x89 /* NEGATIVE, DECIMAL, CARRY */},
		{0x99, 0x00, 0x09 /* DECIMAL, CARRY */, 0x99, 0x89 /* NEGATIVE, DECIMAL, CARRY */},
		{0x99, 0x01, 0x09 /* DECIMAL, CARRY */, 0x98, 0x89 /* NEGATIVE, DECIMAL, CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0x7780
		c.acc = tc.acc
		c.status = tc.status
		c.Write(c.pc, tc.op1)

		if c.SBC(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (status 0x%02x), wanted 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpSEC(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.SEC(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpSED(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.SED(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpSEI(t *testing.T) {
	c := cpu
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
		c.status = tc.status
		c.SEI(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted 0x%02x, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpSTA(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, status      uint8
		want, wantStatus uint8
	}{
		{0x81, 0x80, 0x81, 0x80},
	}

	for i, tc := range cases {
		c.acc = tc.acc
		c.status = tc.status
		c.pc = 0x10 // memory[0x10] should be 0 at init

		c.STA(ZERO_PAGE)

		if v := c.Read(c.getOperandAddr(ZERO_PAGE)); v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpSTX(t *testing.T) {
	c := cpu
	cases := []struct {
		x, status        uint8
		want, wantStatus uint8
	}{
		{0x81, 0x80, 0x81, 0x80},
	}

	for i, tc := range cases {
		c.x = tc.x
		c.status = tc.status
		c.pc = 0x10 // memory[0x10] should be 0 at init

		c.STX(ZERO_PAGE)

		if v := c.Read(c.getOperandAddr(ZERO_PAGE)); v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpSTY(t *testing.T) {
	c := cpu
	cases := []struct {
		y, status        uint8
		want, wantStatus uint8
	}{
		{0x81, 0x80, 0x81, 0x80},
	}

	for i, tc := range cases {
		c.y = tc.y
		c.status = tc.status
		c.pc = 0x10 // memory[0x10] should be 0 at init

		c.STY(ZERO_PAGE)

		if v := c.Read(c.getOperandAddr(ZERO_PAGE)); v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpTAX(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, x     uint8
		wantX      uint8
		wantStatus uint8
	}{
		{0xFF, 0x01, 0xFF, 0x80 /* NEGATIVE */},
		{0x00, 0x01, 0x00, 0x02 /* ZERO */},
	}

	for i, tc := range cases {
		c.acc = tc.acc
		c.x = tc.x
		c.status = 0 // clear

		if c.TAX(IMPLICIT); c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.x, c.status, tc.wantX, tc.wantStatus)
		}
	}
}

func TestOpTAY(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, y     uint8
		wantY      uint8
		wantStatus uint8
	}{
		{0xFF, 0x01, 0xFF, 0x80 /* NEGATIVE */},
		{0x00, 0x01, 0x00, 0x02 /* ZERO */},
	}

	for i, tc := range cases {
		c.acc = tc.acc
		c.y = tc.y
		c.status = 0 // clear

		if c.TAY(IMPLICIT); c.y != tc.wantY || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.x, c.status, tc.wantY, tc.wantStatus)
		}
	}
}

func TestOpTSX(t *testing.T) {
	c := cpu
	cases := []struct {
		sp, x      uint8
		wantX      uint8
		wantStatus uint8
	}{
		{0xFF, 0x01, 0xFF, 0x80 /* NEGATIVE */},
		{0x00, 0x01, 0x00, 0x02 /* ZERO */},
	}

	for i, tc := range cases {
		c.sp = tc.sp
		c.x = tc.x
		c.status = 0 // clear

		if c.TSX(IMPLICIT); c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.x, c.status, tc.wantX, tc.wantStatus)
		}
	}
}

func TestOpTXA(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, x     uint8
		want       uint8
		wantStatus uint8
	}{
		{0xFF, 0x01, 0x01, 0x00},
		{0x00, 0xF1, 0xF1, 0x80 /* NEGATIVE */},
		{0x01, 0x00, 0x00, 0x02 /* ZERO */},
	}

	for i, tc := range cases {
		c.acc = tc.acc
		c.x = tc.x
		c.status = 0 // clear

		if c.TXA(IMPLICIT); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpTXS(t *testing.T) {
	c := cpu
	cases := []struct {
		sp, x, status uint8
		wantSP        uint8
		wantStatus    uint8
	}{
		{0xFF, 0x01, 0x80, 0x01, 0x80},
		{0x01, 0x00, 0x81, 0x00, 0x81},
		{0x01, 0x81, 0x02, 0x81, 0x02},
	}

	for i, tc := range cases {
		c.sp = tc.sp
		c.x = tc.x
		c.status = tc.status

		if c.TXS(IMPLICIT); c.sp != tc.wantSP || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.sp, c.status, tc.wantSP, tc.wantStatus)
		}
	}
}

func TestOpTYA(t *testing.T) {
	c := cpu
	cases := []struct {
		acc, y     uint8
		want       uint8
		wantStatus uint8
	}{
		{0xFF, 0x01, 0x01, 0x00},
		{0x00, 0xF1, 0xF1, 0x80 /* NEGATIVE */},
		{0x01, 0x00, 0x00, 0x02 /* ZERO */},
	}

	for i, tc := range cases {
		c.acc = tc.acc
		c.y = tc.y
		c.status = 0 // clear

		if c.TYA(IMPLICIT); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestFunctionsBin(t *testing.T) {
	tf := "../testdata/6502_functional_test.bin"
	bin, err := os.ReadFile(tf)
	if err != nil {
		t.Errorf("Couldn't read testdata file %q: %v", tf, err)
	}

	c := cpu
	c.Reset()
	c.LoadMem(0x000A, bin)

	c.SetPC(0x0400)

	for {
		prev_pc := c.PC()
		if c.Step(); c.PC() == prev_pc {
			break
		}
	}

	var want uint16 = 0x3469
	if got := c.pc; got != want {
		t.Errorf("PC = 0x%04x, wanted 0x%04x", got, want)
	}
}

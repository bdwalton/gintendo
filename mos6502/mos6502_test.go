package mos6502

import (
	"errors"
	"testing"
)

func memInit(val uint8) (mem [MEM_SIZE + 1]uint8) {
	for i := 0; i <= MEM_SIZE; i++ {
		mem[i] = val
	}
	return
}

func TestMemRead(t *testing.T) {
	c := New()
	cases := []struct {
		mem1 uint8
		want uint8
	}{
		{0xFF, 0xFF},
		{0x11, 0x11},
	}

	for i, tc := range cases {
		c.writeMem(uint16(i), tc.mem1)
		c.pc = uint16(i)
		if got := c.memRead(c.pc); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestMemWrite(t *testing.T) {
	c := New()
	cases := []struct {
		mem1 uint8
		want uint8
	}{
		{0xFF, 0xFF},
		{0x11, 0x11},
	}

	for i, tc := range cases {
		c.pc = uint16(i)
		c.writeMem(c.pc, tc.mem1)
		if got := c.memRead(c.pc); got != tc.want {
			t.Errorf("%d: Got 0x%02x, want 0x%02x", i, got, tc.want)
		}
	}
}

func TestMemRead16(t *testing.T) {
	c := New()
	cases := []struct {
		mem1, mem2 uint8
		want       uint16
	}{
		{0xFF, 0x11, 0x11FF},
		{0xFF, 0x11, 0x11FF},
	}

	for i, tc := range cases {
		c.memory[i] = tc.mem1
		c.memory[i+1] = tc.mem2
		c.pc = uint16(i)
		if got := c.memRead16(c.pc); got != tc.want {
			t.Errorf("%d: Got 0x%04x, want 0x%04x", i, got, tc.want)
		}
	}
}

func TestMemWrite16(t *testing.T) {
	c := New()
	cases := []struct {
		val        uint16
		mem1, mem2 uint8
	}{
		{0x11FF, 0xFF, 0x11},
		{0x5566, 0x66, 0x55},
	}

	for i, tc := range cases {
		c.pc = uint16(i)
		c.writeMem16(c.pc, tc.val)
		c.memory[i] = tc.mem1
		c.memory[i+1] = tc.mem2

		if c.memory[i] != tc.mem1 || c.memory[i+1] != tc.mem2 {
			t.Errorf("%d: Got (0x%02x, 0x%02x), want (0x%02x, 0x%02x)", i, c.memory[i], c.memory[i+1], tc.mem1, tc.mem2)
		}
	}
}

func TestPushAddress(t *testing.T) {
	c := New()
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
		if c.sp != tc.wantSP || c.memRead(c.getStackAddr()+2) != tc.wantHI || c.memRead(c.getStackAddr()+1) != tc.wantLO {
			top := c.getStackAddr() + 2
			bottom := top - 1
			t.Errorf("%d: Got 0x%02x %v, want 0x%02x %v", i, c.sp, c.memRange(bottom, top), tc.wantSP, []uint8{tc.wantLO, tc.wantHI})
		}

	}
}

func TestPopAddress(t *testing.T) {
	c := New()
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
	c := New()

	c.memory[0x0F] = 0x44
	c.memory[0x10] = 0x55
	c.memory[0x64] = 0x0F
	c.memory[0x65] = 0x11
	c.memory[0x001F] = 0x55
	c.memory[0x110F] = 0xFA
	c.memory[0x1110] = 0xBB
	c.memory[0xFF66] = 0x82
	c.x = 0x10
	c.y = 0xAC

	cases := []struct {
		pc   uint16
		mode uint8
		want uint16
	}{
		{0x0064, IMMEDIATE, 0x64},     // Should just return program counter
		{0x0064, ZERO_PAGE, 0x000F},   // mem[pc]
		{0x0064, ZERO_PAGE_X, 0x001F}, // mem[pc] + x
		{0x0064, ZERO_PAGE_Y, 0x00BB}, // mem[pc] + y
		{0x0064, RELATIVE, 0x73},      // pc + int8(mem[pc])
		{0xFF66, RELATIVE, 0xFEE8},    // pc - int8(mem[pc])
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
	c := New()
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
		c.memory[0] = tc.val
		got, err := c.getInst()
		if got != tc.want || (err != nil && tc.wantErr == nil) || !errors.Is(err, tc.wantErr) {
			t.Errorf("%d: got %s, want %s; err %v, wantErr %v", i, got, tc.want, err, tc.wantErr)
		}
	}

}

func TestOpADC(t *testing.T) {
	c := New()
	cases := []struct {
		acc, op1, status uint8
		want, wantStatus uint8
	}{
		{0xFF, 0x01, 0x00, 0x00, 0x03 /* ZERO, CARRY */},
		{0xF1, 0x01, 0x00, 0xF2, 0x80 /* NEGATIVE */},
		{0x00, 0x00, 0x00, 0x00, 0x02 /* ZERO */},
		{0xF0, 0x0F, 0x00, 0xFF, 0x80 /* NEGATIVE */},
		{0xFF, 0xF0, 0x01 /* CARRY */, 0xF0, 0x81 /* NEGATIVE, CARRY */},
		{0xEF, 0xE1, 0x00, 0xD0, 0x81 /* NEGATIVE, CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0x7780
		c.acc = tc.acc
		c.status = tc.status
		c.writeMem(c.pc, tc.op1)

		if c.opADC(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (status 0x%02x), wanted 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpAND(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1
		c.acc = tc.acc

		if c.opAND(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpASL(t *testing.T) {
	c := New()
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
		switch tc.mode {
		case ACCUMULATOR:
			c.acc = tc.val
		default:
			c.writeMem(c.getOperandAddr(tc.mode), tc.val)
		}

		c.opASL(tc.mode)

		var got uint8
		switch tc.mode {
		case ACCUMULATOR:
			got = c.acc
		default:
			got = c.memRead(c.getOperandAddr(tc.mode))
		}
		if got != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, status 0x%02x; Want 0x%02x, status 0x%02x", i, got, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpBCC(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x01 /* CARRY */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x01 /* CARRY */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6681},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBCC(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBCS(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x01 /* CARRY */, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x01 /* CARRY */, 0x6681},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBCS(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBEQ(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x02 /* ZERO */, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x02 /* ZERO */, 0x6681},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBEQ(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBIT(t *testing.T) {
	c := New()
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
		c.acc = tc.acc
		c.writeMem(c.getOperandAddr(ZERO_PAGE), tc.op)

		if c.opBIT(ZERO_PAGE); c.status != tc.wantStatus {
			t.Errorf("%d: Got status = 0x%02x, wanted 0x%02x", i, c.status, tc.wantStatus)
		}
	}
}

func TestOpBMI(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x80 /* NEGATIVE */, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x80 /* NEGATIVE */, 0x6681},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBMI(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBNE(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x02 /* ZERO */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x02 /* ZERO */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6681},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBNE(RELATIVE)

		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBPL(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x80 /* NEGATIVE */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x80 /* NEGATIVE */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6681},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBPL(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBRK(t *testing.T) {
	c := New()
	cases := []struct {
		pc         uint16
		brk        uint16
		status     uint8
		wantPC     uint16
		wantStatus uint8
	}{
		{0xFF15, 0xAC69, 0x00, 0xAC69, 0x10 /* BRK set */},
		{0xAAAA, 0x1167, 0x81, 0x1167, 0x91 /* BRK set */},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem16(INT_BRK, tc.brk)
		c.opBRK(IMPLICIT)
		if c.pc != tc.wantPC || c.status != tc.wantStatus {
			t.Errorf("%d: PC = 0x%04x (status 0x%02x), wanted 0x%04x (status 0x%02x)", i, c.pc, c.status, tc.wantPC, tc.wantStatus)
		}
	}
}

func TestOpBVC(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x40 /* OVERFLOW */, 0x6677},
		{0x6677, 0x0A /* +10 */, 0x40 /* OVERFLOW */, 0x6677},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x00, 0x6681},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBVC(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpBVS(t *testing.T) {
	c := New()
	cases := []struct {
		pc     uint16
		offset uint8
		status uint8
		wantPC uint16
	}{
		{0x6677, 0xF6 /* -10 */, 0x40 /* OVERFLOW */, 0x666D},
		{0x6677, 0x0A /* +10 */, 0x40 /* OVERFLOW */, 0x6681},
		{0x6677, 0xF6 /* -10 */, 0x00, 0x6677},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.status = tc.status
		c.writeMem(c.pc, tc.offset)
		c.opBVS(RELATIVE)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, want 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpCLC(t *testing.T) {
	c := New()
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
		c.opCLC(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCLD(t *testing.T) {
	c := New()
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
		c.opCLD(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCLI(t *testing.T) {
	c := New()
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
		c.opCLI(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpCLV(t *testing.T) {
	c := New()
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
		c.opCLV(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpDEC(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1

		if c.opDEC(IMMEDIATE); c.memory[c.pc] != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpDEX(t *testing.T) {
	c := New()
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
		c.opDEX(IMPLICIT)
		if c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantX, tc.wantStatus, c.x, c.status)
		}
	}
}

func TestOpDEY(t *testing.T) {
	c := New()
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
		c.opDEY(IMPLICIT)
		if c.y != tc.wantY || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantY, tc.wantStatus, c.y, c.status)
		}
	}
}

func TestOpEOR(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1
		c.acc = tc.acc

		if c.opEOR(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpINX(t *testing.T) {
	c := New()
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
		c.opINX(IMPLICIT)
		if c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantX, tc.wantStatus, c.x, c.status)
		}
	}
}

func TestOpINY(t *testing.T) {
	c := New()
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
		c.opINY(IMPLICIT)
		if c.y != tc.wantY || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status: 0x%02x), got %d (status 0x%02x)", i, tc.wantY, tc.wantStatus, c.y, c.status)
		}
	}
}

func TestOpINC(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1

		if c.opINC(IMMEDIATE); c.memory[c.pc] != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.memory[c.pc], c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpJMP(t *testing.T) {
	c := New()
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

		c.writeMem16(c.getOperandAddr(ABSOLUTE), tc.target)
		c.writeMem16(c.getOperandAddr(INDIRECT), tc.target2)

		c.opJMP(tc.mode)
		if c.pc != tc.wantPC {
			t.Errorf("%d: PC = 0x%04x, wanted 0x%04x", i, c.pc, tc.wantPC)
		}
	}
}

func TestOpJSR(t *testing.T) {
	c := New()
	cases := []struct {
		pc               uint16
		target           uint16
		sp               uint8
		wantPC, wantAddr uint16
	}{
		{0x02FF, 0xAC01, 0xFF, 0xAC01, 0x02FE},
		{0x03AB, 0xDD01, 0xFE, 0xDD01, 0x03AA},
	}

	for i, tc := range cases {
		c.pc = tc.pc
		c.writeMem16(c.pc, tc.target)
		c.sp = tc.sp

		c.opJSR(ABSOLUTE)

		if addr := c.popAddress(); c.pc != tc.wantPC || addr != tc.wantAddr {
			t.Errorf("%d: Got PC = 0x%04x, Addr = 0x%04x; Want PC = 0x%04x, Addr = 0x%04x", i, c.pc, addr, tc.wantPC, tc.wantAddr)
		}
	}
}

func TestOpLDA(t *testing.T) {
	c := New()
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
		c.writeMem(c.pc, tc.op1)

		if c.opLDA(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpLDX(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1

		if c.opLDX(IMMEDIATE); c.x != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.x, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpLDY(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1

		if c.opLDY(IMMEDIATE); c.y != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.y, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpLSR(t *testing.T) {
	c := New()
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
		switch tc.mode {
		case ACCUMULATOR:
			c.acc = tc.val
		default:
			c.writeMem(c.getOperandAddr(tc.mode), tc.val)
		}

		c.opLSR(tc.mode)

		var got uint8
		switch tc.mode {
		case ACCUMULATOR:
			got = c.acc
		default:
			got = c.memRead(c.getOperandAddr(tc.mode))
		}
		if got != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x, status 0x%02x; Want 0x%02x, status 0x%02x", i, got, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpNOP(t *testing.T) {
	c := New()
	c.memory = memInit(0xEA) // NOP

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
		c.pc = tc.pc
		c.status = tc.status
		c.step()
		if c.pc != tc.wantPC || c.status != tc.wantStatus {
			t.Errorf("%d: Wanted %d (status 0x%02x), got %d (status: 0x%02x)", i, tc.wantPC, tc.wantStatus, c.pc, c.status)
		}
	}
}

func TestOpORA(t *testing.T) {
	c := New()
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
		c.memory[c.pc] = tc.op1
		c.acc = tc.acc

		if c.opORA(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (0x%02x), want 0x%02x (0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpPHA(t *testing.T) {
	c := New()
	cases := []struct {
		acc    uint8
		wantSP uint8
	}{
		{0x01, 0xFE},
		{0x02, 0xFD},
		{0xFF, 0xFC},
	}

	for i, tc := range cases {
		c.acc = tc.acc
		if c.opPHA(IMPLICIT); c.memory[c.getStackAddr()+1] != tc.acc || c.sp != tc.wantSP {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; Mem = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, c.memory[c.getStackAddr()-1], tc.acc)
		}
	}
}

func TestOpPHP(t *testing.T) {
	c := New()
	cases := []struct {
		status uint8
		wantSP uint8
	}{
		{0x01, 0xFE},
		{0x02, 0xFD},
		{0x80, 0xFC},
	}

	for i, tc := range cases {
		c.status = tc.status
		if c.opPHP(IMPLICIT); c.memory[c.getStackAddr()+1] != tc.status || c.sp != tc.wantSP {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; Mem = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, c.memory[c.sp-1], tc.status)
		}
	}
}

func TestOpPLA(t *testing.T) {
	c := New()
	cases := []struct {
		acc        uint8
		wantSP     uint8
		wantStatus uint8
	}{
		{0xFE, 0xFC, 0x80},
		{0x82, 0xFD, 0x80},
		{0x00, 0xFE, 0x02},
		{0x01, 0xFF, 0x00},
	}

	// Adjust c.sp with these calls, in reverse from the cases
	// we'll compare as we pop.
	for i := len(cases); i > 0; i -= 1 {
		c.acc = cases[i-1].acc
		c.opPHA(IMPLICIT)
	}

	for i, tc := range cases {
		c.acc = 0
		c.status = 0
		if c.opPLA(IMPLICIT); c.sp != tc.wantSP || c.acc != tc.acc || c.status != tc.wantStatus {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; ACC = 0x%02x, want 0x%02x; Status = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, c.acc, tc.acc, c.status, tc.wantStatus)
		}
	}
}

func TestOpPLP(t *testing.T) {
	c := New()
	cases := []struct {
		status     uint8
		wantSP     uint8
		wantStatus uint8
	}{
		{0x80, 0xFC, 0x80},
		{0x81, 0xFD, 0x81},
		{0x00, 0xFE, 0x00},
		{0x01, 0xFF, 0x01},
	}

	// Adjust c.sp with these calls, in reverse from the cases
	// we'll compare as we pop.
	for i := len(cases); i > 0; i -= 1 {
		c.status = cases[i-1].status
		c.opPHP(IMPLICIT)
	}

	for i, tc := range cases {
		c.status = 0
		if c.opPLP(IMPLICIT); c.sp != tc.wantSP || c.status != tc.wantStatus {
			t.Errorf("%d: SP=0x%02x, want 0x%02x; Status = 0x%02x, want 0x%02x", i, c.sp, tc.wantSP, c.status, tc.wantStatus)
		}
	}
}

func TestOpROL(t *testing.T) {
	c := New()
	cases := []struct {
		acc, op1   uint8 // Seeded acc and memory location 0
		mode       uint8 // Addressing mode (ACCUMULATOR or ZERO_PAGE)
		status     uint8 // Current status
		want       uint8 // Value of ACC or OP1 after ROL
		wantStatus uint8 // Value of status after ROL
	}{
		{0x00, 0x00, ACCUMULATOR, 0x00, 0x00, 0x02 /* ZERO */},
		{0x01, 0x00, ACCUMULATOR, 0x00, 0x02, 0x00 /* ZERO */},
		{0x00, 0x00, ACCUMULATOR, 0x01 /* CARRY */, 0x01, 0x00},
		{0x01, 0x01, ACCUMULATOR, 0x01 /* CARRY */, 0x03, 0x00},
		{0x01, 0x01, ACCUMULATOR, 0x00, 0x02, 0x00},
		{0x80, 0x01, ACCUMULATOR, 0x00, 0x01, 0x01 /* CARRY */},
		{0x81, 0x01, ACCUMULATOR, 0x00, 0x03, 0x01 /* CARRY */},
		{0xC1, 0x01, ACCUMULATOR, 0x00, 0x83, 0x81 /* CARRY, NEGATIVE */},
		{0x00, 0x01, ZERO_PAGE, 0x00, 0x02, 0x00},
		{0x00, 0x01, ZERO_PAGE, 0x01 /* CARRY */, 0x03, 0x00},
		{0x01, 0x01, ZERO_PAGE, 0x01 /* CARRY */, 0x03, 0x00},
		{0x01, 0x01, ZERO_PAGE, 0x00, 0x02, 0x00},
		{0x01, 0x80, ZERO_PAGE, 0x00, 0x01, 0x01 /* CARRY */},
		{0x01, 0x81, ZERO_PAGE, 0x00, 0x03, 0x01 /* CARRY */},
		{0x01, 0xC1, ZERO_PAGE, 0x00, 0x83, 0x81 /* CARRY, NEGATIVE */},
	}

	for i, tc := range cases {
		c.pc = 0x10 // memory addr 0x10 should always be 0 on init
		c.acc = tc.acc
		if tc.mode != ACCUMULATOR {
			c.writeMem(c.getOperandAddr(tc.mode), tc.op1)
		}

		c.status = tc.status

		c.opROL(tc.mode)
		v := c.acc
		if tc.mode == ZERO_PAGE {
			v = c.memRead(c.getOperandAddr(tc.mode)) // We don't run step(), so PC isn't updated
		}

		if v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status = 0x%02x), want 0x%02x (status = 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpROR(t *testing.T) {
	c := New()
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
		{0x01, 0x01, ACCUMULATOR, 0x00, 0x80, 0x81},
		{0x80, 0x01, ACCUMULATOR, 0x00, 0x40, 0x00},
		{0x81, 0x01, ACCUMULATOR, 0x00, 0xC0, 0x81 /* NEGATIVE, CARRY */},
		{0xC1, 0x01, ACCUMULATOR, 0x00, 0xE0, 0x81 /* NEGATIVE, CARRY */},
		{0x00, 0x00, ZERO_PAGE, 0x00, 0x00, 0x02 /* ZERO */},
		{0x00, 0x01, ZERO_PAGE, 0x00, 0x80, 0x81 /* NEGATIVE, CARRY */},
		{0x00, 0x02, ZERO_PAGE, 0x01, 0x81, 0x80 /* NEGATIVE */},
		{0x00, 0x01, ZERO_PAGE, 0x01 /* CARRY */, 0x80, 0x81},
		{0x00, 0x81, ZERO_PAGE, 0x00, 0xC0, 0x81 /* NEGATIVE, CARRY */},
		{0x00, 0x82, ZERO_PAGE, 0x01, 0xC1, 0x80 /* NEGATIVE */},
	}

	for i, tc := range cases {
		c.pc = 0x10 // memory addr 0x10 should always be 0 on init
		c.acc = tc.acc
		if tc.mode != ACCUMULATOR {
			c.writeMem(c.getOperandAddr(tc.mode), tc.op1)
		}
		c.status = tc.status

		c.opROR(tc.mode)
		v := c.acc
		if tc.mode == ZERO_PAGE {
			v = c.memRead(c.getOperandAddr(tc.mode)) // We don't run step(), so PC isn't updated
		}

		if v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status = 0x%02x), want 0x%02x (status = 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpRTI(t *testing.T) {
	c := New()
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

		c.opRTI(IMPLICIT)
		if c.pc != tc.wantPC || c.status != tc.wantStatus {
			t.Errorf("%d: PC = 0x%04x (status 0x%02x), wanted 0x%04x (status 0x%02x)", i, c.pc, c.status, tc.wantPC, tc.wantStatus)

		}
	}
}

func TestOpRTS(t *testing.T) {
	c := New()
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

		if c.opRTS(IMPLICIT); c.pc != tc.wantPC || c.sp != tc.wantSP {
			t.Errorf("%d: Got PC = 0x%04x, SP = 0x%02x, want PC = 0x%04x, SP = 0x%02x", i, c.pc, c.sp, tc.wantPC, tc.wantSP)
		}
	}
}

func TestOpSBC(t *testing.T) {
	c := New()
	cases := []struct {
		acc, op1, status uint8
		want, wantStatus uint8
	}{
		{0xFF, 0x01, 0x01, 0xFE, 0x81},
		{0x42, 0x01, 0x01, 0x41, 0x01},
		{0x42, 0x42, 0x01, 0x00, 0x03 /* ZERO, CARRY */},
		{0xD0, 0x70, 0x01, 0x60, 0x41 /* OVERFLOW, CARRY */},
	}

	for i, tc := range cases {
		c.pc = 0x7780
		c.acc = tc.acc
		c.status = tc.status
		c.writeMem(c.pc, tc.op1)

		if c.opSBC(IMMEDIATE); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: Got 0x%02x (status 0x%02x), wanted 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpSEC(t *testing.T) {
	c := New()
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
		c.opSEC(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpSED(t *testing.T) {
	c := New()
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
		c.opSED(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted %d, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpSEI(t *testing.T) {
	c := New()
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
		c.opSEI(IMPLICIT)
		if c.status != tc.want {
			t.Errorf("%d: Wanted 0x%02x, got 0x%02x", i, tc.want, c.status)
		}
	}
}

func TestOpSTA(t *testing.T) {
	c := New()
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

		c.opSTA(ZERO_PAGE)

		if v := c.memRead(c.getOperandAddr(ZERO_PAGE)); v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpSTX(t *testing.T) {
	c := New()
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

		c.opSTX(ZERO_PAGE)

		if v := c.memRead(c.getOperandAddr(ZERO_PAGE)); v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpSTY(t *testing.T) {
	c := New()
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

		c.opSTY(ZERO_PAGE)

		if v := c.memRead(c.getOperandAddr(ZERO_PAGE)); v != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, v, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpTAX(t *testing.T) {
	c := New()
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

		if c.opTAX(IMPLICIT); c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.x, c.status, tc.wantX, tc.wantStatus)
		}
	}
}

func TestOpTAY(t *testing.T) {
	c := New()
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

		if c.opTAY(IMPLICIT); c.y != tc.wantY || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.x, c.status, tc.wantY, tc.wantStatus)
		}
	}
}

func TestOpTSX(t *testing.T) {
	c := New()
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

		if c.opTSX(IMPLICIT); c.x != tc.wantX || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.x, c.status, tc.wantX, tc.wantStatus)
		}
	}
}

func TestOpTXA(t *testing.T) {
	c := New()
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

		if c.opTXA(IMPLICIT); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

func TestOpTXS(t *testing.T) {
	c := New()
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

		if c.opTXS(IMPLICIT); c.sp != tc.wantSP || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.sp, c.status, tc.wantSP, tc.wantStatus)
		}
	}
}

func TestOpTYA(t *testing.T) {
	c := New()
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

		if c.opTYA(IMPLICIT); c.acc != tc.want || c.status != tc.wantStatus {
			t.Errorf("%d: got 0x%02x (status 0x%02x), want 0x%02x (status 0x%02x)", i, c.acc, c.status, tc.want, tc.wantStatus)
		}
	}
}

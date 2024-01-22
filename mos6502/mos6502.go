// Package mos6502 implements the MOS Technologies 6502 processor.
package mos6502

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
)

const (
	MAX_ADDRESS = math.MaxUint16
	MEM_SIZE    = MAX_ADDRESS + 1
)

// 6502 Interrupt Vectors
// https://en.wikipedia.org/wiki/Interrupts_in_65xx_processors
const (
	INT_NONE  = 0x0000 // A dummy value
	INT_IRQ   = 0xFFFE
	INT_BRK   = INT_IRQ
	INT_RESET = 0xFFFC
	INT_NMI   = 0xFFFA
)

// 6502 Processor Status Flags
// https://www.nesdev.org/obelisk-6502-guide/registers.html
const (
	STATUS_FLAG_CARRY             = 1 << 0 // C
	STATUS_FLAG_ZERO              = 1 << 1 // Z
	STATUS_FLAG_INTERRUPT_DISABLE = 1 << 2 // I
	STATUS_FLAG_DECIMAL           = 1 << 3 // D
	STATUS_FLAG_BREAK             = 1 << 4 // B
	UNUSED_STATUS_FLAG            = 1 << 5 // This is never used but is always on
	STATUS_FLAG_OVERFLOW          = 1 << 6 // V
	STATUS_FLAG_NEGATIVE          = 1 << 7 // N
)

var flagMap map[uint8]byte = map[uint8]byte{
	STATUS_FLAG_CARRY:             'C',
	STATUS_FLAG_ZERO:              'Z',
	STATUS_FLAG_INTERRUPT_DISABLE: 'I',
	STATUS_FLAG_DECIMAL:           'D',
	STATUS_FLAG_BREAK:             'B',
	UNUSED_STATUS_FLAG:            '-',
	STATUS_FLAG_OVERFLOW:          'V',
	STATUS_FLAG_NEGATIVE:          'N',
}

func statusString(p uint8) string {
	var sb strings.Builder

	flags := []uint8{
		STATUS_FLAG_NEGATIVE,
		STATUS_FLAG_OVERFLOW,
		UNUSED_STATUS_FLAG,
		STATUS_FLAG_BREAK,
		STATUS_FLAG_DECIMAL,
		STATUS_FLAG_INTERRUPT_DISABLE,
		STATUS_FLAG_ZERO,
		STATUS_FLAG_CARRY,
	}

	for _, f := range flags {
		if p&f > 0 {
			sb.WriteByte(flagMap[f])
		} else {
			sb.WriteByte('.')
		}
	}

	return sb.String()
}

// Type Bus is how we'll abstract memory read/write outside of cpu registers.
type Bus interface {
	Read(uint16) uint8
	Write(uint16, uint8)
}

// Type CPU implements all of the machine state for the 6502
type CPU struct {
	acc              uint8  // main register
	x, y             uint8  // index registers
	status           uint8  // a register for storing various status bits
	sp               uint8  // stack pointer - stack is 0x0100-0x01FF so only 8 bits needed
	pc               uint16 // the program counter
	mem              Bus    // 64k addressable memory, often backed by a mapper.
	cycles           int    // how many cycles an instruction consumes
	pendingInterrupt int    // 0/INTERRUPT_NONE, INTERRUPT_NMI or INTERRUPT_IRQ
	nmiTriggered     bool   // Set when NMI was triggered so we know to account for cycles
}

func (c *CPU) String() string {
	return fmt.Sprintf("A,X,Y: 0x%02x, 0x%02x, 0x%02x; PC: 0x%04x, SP: 0x%02x, P: %s; OP: %s", c.acc, c.x, c.y, c.pc, c.sp, statusString(c.status), opcodes[c.mem.Read(c.pc)])
}

func New(b Bus) *CPU {
	// Power on state values from:
	// https://nesdev-wiki.nes.science/wikipages/CPU_ALL.xhtml#Power_up_state
	// B is not normally visible in the register, but per docs, is
	// set at startup.
	c := &CPU{
		sp:     0xFD,
		mem:    b,
		status: UNUSED_STATUS_FLAG | STATUS_FLAG_BREAK | STATUS_FLAG_INTERRUPT_DISABLE,
	}
	c.pc = c.Read16(INT_RESET, ABSOLUTE)
	return c
}

var invalidInstruction = errors.New("invalid instruction")

func (c *CPU) getInst() (opcode, error) {
	m := c.mem.Read(c.pc)
	op, ok := opcodes[m]
	if !ok {
		return opcode{}, fmt.Errorf("pc: 0x%04x, inst: 0x%02x - %w", c.pc, m, invalidInstruction)
	}

	return op, nil
}

// memRange returns a slice of memory addresses from low to
// high. Mostly useful for debugging.
func (c *CPU) memRange(low, high uint16) []uint8 {
	ret := make([]uint8, high-low)
	for i := low; i <= high; i += 1 {
		ret = append(ret, c.mem.Read(uint16(i)))
	}

	return ret
}

// Read16 returns the two bytes from memory at addr (lower byte is
// first). The mode parameter helps handle wrapping cases in certain
// usecases.
func (c *CPU) Read16(addr uint16, mode uint8) uint16 {
	lsb := uint16(c.mem.Read(addr))

	addr++

	if mode == INDIRECT_X || mode == INDIRECT_Y { // handle wrapping
		addr &= 0x00FF
	}

	msb := uint16(c.mem.Read(addr))

	return (msb << 8) | lsb
}

// getOperandAddr takes a mode and returns an address for the operand
// referenced by the program counter. It assumes that the counter was
// incremented past the actual instruction itself.
func (c *CPU) getOperandAddr(mode uint8) uint16 {
	var addr uint16
	switch mode {
	case ACCUMULATOR:
		panic("ACCUMULATOR Address mode should never use this method")
	case IMPLICIT:
		panic("IMPLICIT Address mode should never use this method")
	case IMMEDIATE:
		addr = c.pc
	case ZERO_PAGE:
		addr = uint16(c.mem.Read(c.pc))
	case ZERO_PAGE_X:
		return uint16(c.mem.Read(c.pc) + c.x)
	case ZERO_PAGE_Y, ZERO_PAGE_X_BUT_Y:
		return uint16(c.mem.Read(c.pc) + c.y)
	case ABSOLUTE:
		return c.Read16(c.pc, mode)
	case ABSOLUTE_X:
		a := c.Read16(c.pc, mode)
		addr = a + uint16(c.x)
		c.cycles += extraCycles(a, addr)
	case ABSOLUTE_Y:
		a := c.Read16(c.pc, mode)
		addr = a + uint16(c.y)
		c.cycles += extraCycles(a, addr)
	case INDIRECT:
		return c.Read16(c.Read16(c.pc, mode), mode)
	case INDIRECT_X:
		return c.Read16(uint16(c.mem.Read(c.pc)+c.x), mode)
	case INDIRECT_Y:
		a := c.Read16(uint16(c.mem.Read(c.pc)), mode)
		addr = a + uint16(c.y)
		c.cycles += extraCycles(a, addr)
	case RELATIVE:
		// Relative from PC at time of instruction
		// execution. We advance pc as soon as we eat the byte
		// from memory to decode the instruction, so we need
		// to account for that here and step over the relative
		// argument while calculating the new target address.
		addr = (c.pc + 1) + uint16(int8(c.mem.Read(c.pc)))
	default:
		panic("Invalid addressing mode")
	}

	return addr
}

// Write16 stores val at addr (lower byte is first).
func (c *CPU) Write16(addr, val uint16) {
	c.mem.Write(addr, uint8(val&0x00FF))
	c.mem.Write(addr+1, uint8(val>>8))
}

func (c *CPU) TriggerNMI() {
	c.pendingInterrupt = INT_NMI
}

func (c *CPU) TriggerIRQ() {
	if c.status&STATUS_FLAG_INTERRUPT_DISABLE == 0 {
		c.pendingInterrupt = INT_IRQ
	}
}

func (c *CPU) AddDMACycles() {
	// TODO: Handle the extra cycle that might occur depending on
	// timing of when the DMA call is triggered.
	c.cycles += 513
}

func (c *CPU) Reset() {
	// Reset is the only time we should ever touch the unused flag
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE | UNUSED_STATUS_FLAG)
	c.pc = c.Read16(INT_RESET, ABSOLUTE)
	c.cycles = 0
}

// PC returns the current value of the program counter
func (c *CPU) PC() uint16 {
	return c.pc
}

// SetPC will set the program counter to the specified address
func (c *CPU) SetPC(addr uint16) {
	c.pc = addr
}

// Inst returns a string version of the current instruction. Useful
// for debugging utilities or (eg) a BIOS loop.
func (c *CPU) Inst() string {
	var sb strings.Builder
	op := opcodes[c.mem.Read(c.pc)]
	for i := 0; i < int(op.bytes); i++ {
		m := c.pc + uint16(i)
		sb.WriteString(fmt.Sprintf("%04x: 0x%02x ", m, c.mem.Read(m)))
	}
	return sb.String()
}

// LoadMem will write out mem to the CPU's memory, starting at address
// 'start'.
func (c *CPU) LoadMem(start uint16, mem []uint8) {
	for i, m := range mem {
		c.mem.Write(start+uint16(i), m)
	}
}

// Tick should be called by the system bus at machine frequency. It
// will only execute a CPU instruction when we've paid down the cycle
// debt from the last one.
func (c *CPU) Tick() {
	if c.cycles > 0 {
		c.cycles -= 1
		return
	}

	c.Step()
}

// Step will single step the CPU forward, returning the number of
// cycles consumed to complete the execution of the instruction. It
// executes the current instruction (at PC) and advances PC when
// finished.
func (c *CPU) Step() int {
	if c.pendingInterrupt != INT_NONE {
		c.pushAddress(c.pc)
		c.pushStack(c.status)
		c.pc = c.Read16(uint16(c.pendingInterrupt), ABSOLUTE)
		c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE)
		switch c.pendingInterrupt {
		case INT_NMI:
			c.cycles = 7
		case INT_IRQ:
			c.cycles = 8
		}

		c.pendingInterrupt = INT_NONE
		return c.cycles
	}

	op, err := c.getInst()
	if err != nil {
		panic(err)
	}

	c.cycles += int(op.cycles)
	c.pc += 1
	opc := c.pc

	v := reflect.ValueOf(c)
	v.MethodByName(op.name).Call([]reflect.Value{reflect.ValueOf(op.mode)})

	// If we didn't branch, move the PC beyond the full width of
	// the instruction. We consumed the first byte for the
	// instruction code, so only skip over the remaining argument
	// bytes.
	if c.pc == opc {
		c.pc += uint16(op.bytes) - 1
	}

	return c.cycles
}

// setNegativeAndZeroFlags sets the STATUS_FLAG_NEGATIVE and
// STATUS_FLAG_ZERO bits of the status register accordingly for the
// value specified in n.
func (c *CPU) setNegativeAndZeroFlags(n uint8) {
	c.flagsOff(STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	if n == 0 {
		c.flagsOn(STATUS_FLAG_ZERO)
	}

	// SFN is convenitently the same bitmask we'd use to check the
	// msb in a uint8.
	if n&STATUS_FLAG_NEGATIVE != 0 {
		c.flagsOn(STATUS_FLAG_NEGATIVE)
	}
}

// StackAddr returns the address of the stack pointer.
func (c *CPU) StackAddr() uint16 {
	return STACK_PAGE + uint16(c.sp)
}

func (c *CPU) pushStack(val uint8) {
	c.mem.Write(c.StackAddr(), val)
	c.sp -= 1
}

func (c *CPU) popStack() uint8 {
	c.sp += 1
	return c.mem.Read(c.StackAddr())
}

func (c *CPU) pushAddress(addr uint16) {
	c.pushStack(uint8(addr >> 8))     // high
	c.pushStack(uint8(addr & 0x00FF)) // low
}

func (c *CPU) popAddress() uint16 {
	return uint16(c.popStack()) | (uint16(c.popStack()) << 8)
}

// flagsOn forces the flags in mask (STATUS_FLAG_XXX|STATUS_FLAG_YYY)
// on in the status register.
func (c *CPU) flagsOn(mask uint8) {
	c.status = c.status | mask
}

// flagsOff forces the flags in mask (STATUS_FLAG_XXX|STATUS_FLAG_YYY)
// off in the status register.
func (c *CPU) flagsOff(mask uint8) {
	c.status = c.status &^ mask
}

// extraCycles returns 0 if addr1 and add2 are in the same page, 1
// otherwise. This is useful for instructions that take a variable
// number of cycles, depending on whether or not a page boundary is
// crossed.
func extraCycles(addr1, addr2 uint16) int {
	if addr1&0xFF00 != addr2&0xFF00 {
		return 1
	}
	return 0
}

// branch will adjust the PC conditionally based on whether the mask
// bits are set and the resulting comparison is expected to be true or
// false. This allows you to check for STATUS_FLAG being set or
// cleared by: branch(STATUS_FLAG_OVERFLOW, false) -> branch
// when OVERFLOW not set.
func (c *CPU) branch(mask uint8, predicate bool) {
	if (c.status&mask > 0) == predicate {
		a := c.getOperandAddr(RELATIVE)
		// Branching instructions take an extra cycle if they
		// cause a page break. We use pc-1 because we
		// increment it right after reading the op, but that's
		// where we branch from so that's the address we
		// compare to see if we've jumped to a new page.
		c.cycles += extraCycles(a, c.pc-1)
		c.cycles += 1 // successful branches take an extra cycle
		c.pc = a
	}
}

func encodeBCD(val uint8) uint8 {
	return ((uint8(val / 10)) << 4) + (uint8(val % 10))
}

func decodeBCD(val uint8) uint8 {
	return uint8((val>>4)*10) + (val & 0x0F)
}

func (c *CPU) addBCD(val uint8) {
	var res int16
	res = int16(decodeBCD(c.acc)) + int16(decodeBCD(val)) + int16(c.status&STATUS_FLAG_CARRY)
	c.flagsOff(STATUS_FLAG_CARRY)
	if res > 99 {
		res -= 100
		c.flagsOn(STATUS_FLAG_CARRY)
	}
	c.acc = encodeBCD(uint8(res))
	c.setNegativeAndZeroFlags(c.acc)
}

// addWithOverflow adds b to c.acc handling overflow, carry and ZN
// flag setting as appropriate.
func (c *CPU) addWithOverflow(b uint8) {
	res16 := uint16(c.acc) + uint16(b) + uint16(c.status&STATUS_FLAG_CARRY)
	res := uint8(res16)

	var mask uint8
	if (res16 & 0x100) != 0 {
		mask = mask | STATUS_FLAG_CARRY
	}
	if (c.acc^res)&(b^res)&0x80 != 0 {
		mask = mask | STATUS_FLAG_OVERFLOW
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_OVERFLOW | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.flagsOn(mask)

	c.acc = res
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) subBCD(val uint8) {
	var res int16
	res = int16(decodeBCD(c.acc)) - int16(decodeBCD(val))
	if c.status&STATUS_FLAG_CARRY == 0 {
		res -= 1
	}

	c.flagsOn(STATUS_FLAG_CARRY)
	if res < 0 {
		res = 100 - (-1 * res)
		c.flagsOff(STATUS_FLAG_CARRY)
	}

	c.acc = encodeBCD(uint8(res))
	c.setNegativeAndZeroFlags(c.acc)
}

// baseCMP does comparison operations on a and b, setting flags
// accordingly.
func (c *CPU) baseCMP(a, b uint8) {
	c.flagsOff(STATUS_FLAG_ZERO | STATUS_FLAG_NEGATIVE | STATUS_FLAG_CARRY)
	c.setNegativeAndZeroFlags(a - b)
	if a >= b {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *CPU) useDecimalMode() bool {
	return c.status&STATUS_FLAG_DECIMAL != 0
}

func (c *CPU) ADC(mode uint8) {
	v := c.mem.Read(c.getOperandAddr(mode))
	switch c.useDecimalMode() {
	case false:
		c.addWithOverflow(v)
	default:
		c.addBCD(v)

	}
}

func (c *CPU) AND(mode uint8) {
	c.acc = c.acc & c.mem.Read(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) ASL(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = c.acc << 1
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.mem.Read(addr)
		nv = ov << 1
		c.mem.Write(addr, nv)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&0x80 != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *CPU) BCC(mode uint8) {
	c.branch(STATUS_FLAG_CARRY, false)
}

func (c *CPU) BCS(mode uint8) {
	c.branch(STATUS_FLAG_CARRY, true)
}

func (c *CPU) BEQ(mode uint8) {
	c.branch(STATUS_FLAG_ZERO, true)
}

func (c *CPU) BIT(mode uint8) {
	o := c.mem.Read(c.getOperandAddr(mode))

	c.flagsOff(STATUS_FLAG_NEGATIVE | STATUS_FLAG_OVERFLOW | STATUS_FLAG_ZERO)
	var flags uint8
	if (o & c.acc) == 0 {
		flags = flags | STATUS_FLAG_ZERO
	}
	flags = flags | (o & (STATUS_FLAG_NEGATIVE | STATUS_FLAG_OVERFLOW))

	c.flagsOn(flags)
}

func (c *CPU) BMI(mode uint8) {
	c.branch(STATUS_FLAG_NEGATIVE, true)
}

func (c *CPU) BNE(mode uint8) {
	c.branch(STATUS_FLAG_ZERO, false)
}

func (c *CPU) BPL(mode uint8) {
	c.branch(STATUS_FLAG_NEGATIVE, false)
}

func (c *CPU) BRK(mode uint8) {
	// BRK is 2 bytes
	c.pushAddress(c.pc + 1)
	c.pushStack(c.status | STATUS_FLAG_BREAK)
	c.pc = c.Read16(INT_BRK, ABSOLUTE)
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *CPU) BVC(mode uint8) {
	c.branch(STATUS_FLAG_OVERFLOW, false)
}

func (c *CPU) BVS(mode uint8) {
	c.branch(STATUS_FLAG_OVERFLOW, true)
}

func (c *CPU) CLC(mode uint8) {
	c.flagsOff(STATUS_FLAG_CARRY)
}

func (c *CPU) CLD(mode uint8) {
	c.flagsOff(STATUS_FLAG_DECIMAL)
}

func (c *CPU) CLI(mode uint8) {
	c.flagsOff(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *CPU) CLV(mode uint8) {
	c.flagsOff(STATUS_FLAG_OVERFLOW)
}

func (c *CPU) CMP(mode uint8) {
	c.baseCMP(c.acc, c.mem.Read(c.getOperandAddr(mode)))
}

func (c *CPU) CPX(mode uint8) {
	c.baseCMP(c.x, c.mem.Read(c.getOperandAddr(mode)))
}

func (c *CPU) CPY(mode uint8) {
	c.baseCMP(c.y, c.mem.Read(c.getOperandAddr(mode)))
}

func (c *CPU) DEC(mode uint8) {
	a := c.getOperandAddr(mode)
	c.mem.Write(a, c.mem.Read(a)-1)
	c.setNegativeAndZeroFlags(c.mem.Read(a))
}

func (c *CPU) DEX(mode uint8) {
	c.x -= 1
	c.setNegativeAndZeroFlags(c.x)
}

func (c *CPU) DEY(mode uint8) {
	c.y -= 1
	c.setNegativeAndZeroFlags(c.y)
}

func (c *CPU) EOR(mode uint8) {
	c.acc = c.acc ^ c.mem.Read(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) INC(mode uint8) {
	a := c.getOperandAddr(mode)
	c.mem.Write(a, c.mem.Read(a)+1)
	c.setNegativeAndZeroFlags(c.mem.Read(a))
}

func (c *CPU) INX(mode uint8) {
	c.x += 1
	c.setNegativeAndZeroFlags(c.x)
}

func (c *CPU) INY(mode uint8) {
	c.y += 1
	c.setNegativeAndZeroFlags(c.y)
}

func (c *CPU) JMP(mode uint8) {
	c.pc = c.getOperandAddr(mode)
}

func (c *CPU) JSR(mode uint8) {
	c.pushAddress(c.pc + 1) // this is the second byte of the JSR argument
	c.pc = c.getOperandAddr(mode)
}

func (c *CPU) LDA(mode uint8) {
	c.acc = c.mem.Read(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) LDX(mode uint8) {
	c.x = c.mem.Read(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.x)
}

func (c *CPU) LDY(mode uint8) {
	c.y = c.mem.Read(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.y)
}

func (c *CPU) LSR(mode uint8) {
	var ov, nv uint8
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = c.acc >> 1
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.mem.Read(addr)
		nv = ov >> 1
		c.mem.Write(addr, nv)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&STATUS_FLAG_CARRY != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}

}

func (c *CPU) NOP(mode uint8) {
	return
}

func (c *CPU) ORA(mode uint8) {
	c.acc = c.acc | c.mem.Read(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) PHA(mode uint8) {
	c.pushStack(c.acc)
}

func (c *CPU) PHP(mode uint8) {
	// 6502 always sets BREAK when pushing the status register to
	// the stack
	c.pushStack(c.status | STATUS_FLAG_BREAK)
}

func (c *CPU) PLA(mode uint8) {
	c.acc = c.popStack()
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) PLP(mode uint8) {
	c.status = c.popStack() & ^uint8(STATUS_FLAG_BREAK)
	c.flagsOn(UNUSED_STATUS_FLAG)
}

func (c *CPU) ROL(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = (c.acc << 1) | (c.status & STATUS_FLAG_CARRY)
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.mem.Read(addr)
		c.mem.Write(addr, ((ov << 1) | (c.status & STATUS_FLAG_CARRY)))
		nv = c.mem.Read(addr)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	if ov&STATUS_FLAG_NEGATIVE != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
	c.setNegativeAndZeroFlags(nv)
}

func (c *CPU) ROR(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = ov>>1 | ((c.status & STATUS_FLAG_CARRY) << 7)
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.mem.Read(addr)
		c.mem.Write(addr, ((ov >> 1) | ((c.status & STATUS_FLAG_CARRY) << 7)))
		nv = c.mem.Read(addr)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&STATUS_FLAG_CARRY != 0 { // was carry bit set in the old _value_?
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *CPU) RTI(mode uint8) {
	c.status = c.popStack()
	c.pc = c.popAddress()
}

func (c *CPU) RTS(mode uint8) {
	c.pc = c.popAddress() + 1
}

func (c *CPU) SBC(mode uint8) {
	v := c.mem.Read(c.getOperandAddr(mode))
	if c.useDecimalMode() {
		c.subBCD(v)
	} else {
		c.addWithOverflow(^v)
	}
}

func (c *CPU) SEC(mode uint8) {
	c.flagsOn(STATUS_FLAG_CARRY)
}

func (c *CPU) SED(mode uint8) {
	c.flagsOn(STATUS_FLAG_DECIMAL)
}

func (c *CPU) SEI(mode uint8) {
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *CPU) STA(mode uint8) {
	c.mem.Write(c.getOperandAddr(mode), c.acc)
}

func (c *CPU) STX(mode uint8) {
	c.mem.Write(c.getOperandAddr(mode), c.x)
}

func (c *CPU) STY(mode uint8) {
	c.mem.Write(c.getOperandAddr(mode), c.y)
}

func (c *CPU) TAX(mode uint8) {
	c.x = c.acc
	c.setNegativeAndZeroFlags(c.x)
}

func (c *CPU) TAY(mode uint8) {
	c.y = c.acc
	c.setNegativeAndZeroFlags(c.y)
}

func (c *CPU) TSX(mode uint8) {
	c.x = c.sp
	c.setNegativeAndZeroFlags(c.x)
}

func (c *CPU) TXA(mode uint8) {
	c.acc = c.x
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *CPU) TXS(mode uint8) {
	c.sp = c.x
}

func (c *CPU) TYA(mode uint8) {
	c.acc = c.y
	c.setNegativeAndZeroFlags(c.acc)
}

// Undocumented op codes below

func (c *CPU) LAX(mode uint8) {
	m := c.mem.Read(c.getOperandAddr(mode))
	c.acc = m
	c.x = m
}

func (c *CPU) SAX(mode uint8) {
	// TODO: Handle carry flag here. Overflow ignored. Carry not used during subtraction.
	c.x = (c.acc & c.x) - c.mem.Read(c.getOperandAddr(mode))
}

func (c *CPU) DCM(mode uint8) {
	addr := c.getOperandAddr(mode)
	v := c.mem.Read(addr)
	v--
	c.mem.Write(addr, v)
	c.baseCMP(c.acc, v)
}

func (c *CPU) ISB(mode uint8) {
	addr := c.getOperandAddr(mode)
	c.mem.Write(addr, c.mem.Read(addr)+1)
	c.SBC(mode)
}

// Package mos6502 implements the MOS Technologies 6502 processor
// https://en.wikipedia.org/wiki/MOS_Technology_6502
package mos6502

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/bdwalton/gintendo/mappers"
	"strings"
)

const (
	RAM_SIZE = 0x0800 // 2k of real (non-cartride memory)
)

// 6502 Interrupt Vectors
// https://en.wikipedia.org/wiki/Interrupts_in_65xx_processors
const (
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

// 6502 Addressing Modes
// https://www.nesdev.org/obelisk-6502-guide/addressing.html
const (
	IMPLICIT = iota
	ACCUMULATOR
	IMMEDIATE
	ZERO_PAGE
	ZERO_PAGE_X
	ZERO_PAGE_Y
	RELATIVE
	ABSOLUTE
	ABSOLUTE_X
	ABSOLUTE_Y
	INDIRECT
	INDIRECT_X // Indexed Indirect
	INDIRECT_Y // Indirect Indexed
)

const STACK_PAGE = 0x0100

var modenames map[uint8]string = map[uint8]string{IMPLICIT: "IMPLICIT", ACCUMULATOR: "ACCUMULATOR", IMMEDIATE: "IMMEDIATE", ZERO_PAGE: "ZERO_PAGE", ZERO_PAGE_X: "ZERO_PAGE_X", ZERO_PAGE_Y: "ZERO_PAGE_Y", RELATIVE: "RELATIVE", ABSOLUTE: "ABSOLUTE", ABSOLUTE_X: "ABSOLUTE_X", ABSOLUTE_Y: "ABSOLUTE_Y", INDIRECT: "INDIRECT", INDIRECT_X: "INDIRECT_X", INDIRECT_Y: "INDIRECT_Y"}

// 6502 Instructions
// https://www.nesdev.org/obelisk-6502-guide/instructions.html
// https://www.nesdev.org/obelisk-6502-guide/reference.html
const (
	ADC = iota // ADD with Carry
	AND        // Logical AND
	ASL        // Arithmetic Shift Left
	BCC        // Branch if Carry Clear
	BCS        // Branch if Carry Set
	BEQ        // Branch if Equal
	BIT        // Bit Test
	BMI        // Branch if Minus
	BNE        // Branch if Not Equal
	BPL        // Branch if Positive
	BRK        // Force Interrupt
	BVC        // Branch if Overflow Clear
	BVS        // Branch if Overflow Set
	CLC        // Clear Carry Flag
	CLD        // Clear Decimal Mode
	CLI        // Clear Interrupt Disable
	CLV        // Clear Overflow Flag
	CMP        // Compare
	CPX        // Compare X Register
	CPY        // compare Y Regsiter
	DEC        // Decrement Memory
	DEX        // Decrement X Register
	DEY        // Decrement Y Register
	EOR        // Exclusive OR
	INC        // Increment Memory
	INX        // Increment X Register
	INY        // Increment Y Register
	JMP        // Jump
	JSR        // Jump to Subroutine
	LDA        // Load Accumulator
	LDX        // Load X Register
	LDY        // Load Y Register
	LSR        // Logical Shift Right
	NOP        // No Operation
	ORA        // Logical Inclusive OR
	PHA        // Push Accumulator
	PHP        // Push Processor Status
	PLA        // Pull Accumulator
	PLP        // Pull Processor Status
	ROL        // Rotate Left
	ROR        // Rotate Right
	RTI        // Return from Interrupt
	RTS        // Return from Subroutine
	SBC        // Subtract With Carry
	SEC        // Set Carry Flag
	SED        // Set Decimal Flag
	SEI        // Set Interrupt Disable
	STA        // Store Accumulator
	STX        // Store X Register
	STY        // Store Y Register
	TAX        // Transfer Accumulator to X
	TAY        // Transfer Accumulator to Y
	TSX        // Transfer Stack Pointer to X
	TXA        // Transfer X to Accumulator
	TXS        // Transfer X to Stack Pointer
	TYA        // Transfer Y to Accumulator
)

type opcode struct {
	inst   uint8 // The instruction id
	name   string
	mode   uint8 // The memory addressing mode to use
	bytes  uint8 // The number of bytes consumed by operands
	cycles uint8 // The number of cycles consumed by the instruction
}

func (o opcode) String() string {
	return fmt.Sprintf("{%s, %s}", o.name, modenames[o.mode])
}

var opcodes map[uint8]opcode = map[uint8]opcode{
	// ADC
	0x69: opcode{ADC, "ADC", IMMEDIATE, 2, 2},
	0x65: opcode{ADC, "ADC", ZERO_PAGE, 2, 3},
	0x75: opcode{ADC, "ADC", ZERO_PAGE_X, 2, 4},
	0x6D: opcode{ADC, "ADC", ABSOLUTE, 3, 4},
	0x7D: opcode{ADC, "ADC", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x79: opcode{ADC, "ADC", ABSOLUTE_Y, 3, 4 /* +1 if page crossed*/},
	0x61: opcode{ADC, "ADC", INDIRECT_X, 2, 6},
	0x71: opcode{ADC, "ADC", INDIRECT_Y, 2, 5 /* +1 if page crossed*/},
	0x29: opcode{AND, "AND", IMMEDIATE, 2, 2},
	0x25: opcode{AND, "AND", ZERO_PAGE, 2, 3},
	0x35: opcode{AND, "AND", ZERO_PAGE_X, 2, 4},
	0x2D: opcode{AND, "AND", ABSOLUTE, 3, 4},
	0x3D: opcode{AND, "AND", ABSOLUTE_X, 3, 4 /* + 1 if page crossed*/},
	0x39: opcode{AND, "AND", ABSOLUTE_Y, 3, 4 /* +1 if page crossed*/},
	0x21: opcode{AND, "AND", INDIRECT_X, 2, 6},
	0x31: opcode{AND, "AND", INDIRECT_Y, 2, 5 /* +1 if page crossed*/},
	0x0A: opcode{ASL, "ASL", ACCUMULATOR, 1, 2},
	0x06: opcode{ASL, "ASL", ZERO_PAGE, 2, 5},
	0x16: opcode{ASL, "ASL", ZERO_PAGE_X, 2, 6},
	0x0E: opcode{ASL, "ASL", ABSOLUTE, 3, 6},
	0x1E: opcode{ASL, "ASL", ABSOLUTE_X, 3, 7},
	0x90: opcode{BCC, "BCC", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0xB0: opcode{BCS, "BCS", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0xF0: opcode{BEQ, "BEQ", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0x24: opcode{BIT, "BIT", ZERO_PAGE, 2, 3},
	0x2C: opcode{BIT, "BIT", ABSOLUTE, 3, 4},
	0x30: opcode{BMI, "BMI", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0xD0: opcode{BNE, "BNE", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0x10: opcode{BPL, "BPL", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0x00: opcode{BRK, "BRK", IMPLICIT, 2, 7},
	0x50: opcode{BVC, "BVC", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0x70: opcode{BVS, "BVS", RELATIVE, 2, 2 /* +1 if branch succeeds +2 if to a new page */},
	0x18: opcode{CLC, "CLC", IMPLICIT, 1, 2},
	0xD8: opcode{CLD, "CLD", IMPLICIT, 1, 2},
	0x58: opcode{CLI, "CLI", IMPLICIT, 1, 2},
	0xB8: opcode{CLV, "CLV", IMPLICIT, 1, 2},
	0xC9: opcode{CMP, "CMP", IMMEDIATE, 2, 2},
	0xC5: opcode{CMP, "CMP", ZERO_PAGE, 2, 3},
	0xD5: opcode{CMP, "CMP", ZERO_PAGE_X, 2, 4},
	0xCD: opcode{CMP, "CMP", ABSOLUTE, 3, 4},
	0xDD: opcode{CMP, "CMP", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0xD9: opcode{CMP, "CMP", ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xC1: opcode{CMP, "CMP", INDIRECT_X, 2, 6},
	0xD1: opcode{CMP, "CMP", INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	0xE0: opcode{CPX, "CPX", IMMEDIATE, 2, 2},
	0xE4: opcode{CPX, "CPX", ZERO_PAGE, 2, 3},
	0xEC: opcode{CPX, "CPX", ABSOLUTE, 3, 4},
	0xC0: opcode{CPY, "CPY", IMMEDIATE, 2, 2},
	0xC4: opcode{CPY, "CPY", ZERO_PAGE, 2, 3},
	0xCC: opcode{CPY, "CPY", ABSOLUTE, 3, 4},
	0xC6: opcode{DEC, "DEC", ZERO_PAGE, 2, 5},
	0xD6: opcode{DEC, "DEC", ZERO_PAGE_X, 2, 6},
	0xCE: opcode{DEC, "DEC", ABSOLUTE, 3, 6},
	0xDE: opcode{DEC, "DEC", ABSOLUTE_X, 3, 7},
	0xCA: opcode{DEX, "DEX", IMPLICIT, 1, 2},
	0x88: opcode{DEY, "DEY", IMPLICIT, 1, 2},
	0x49: opcode{EOR, "EOR", IMMEDIATE, 2, 2},
	0x45: opcode{EOR, "EOR", ZERO_PAGE, 2, 3},
	0x55: opcode{EOR, "EOR", ZERO_PAGE_X, 2, 4},
	0x4D: opcode{EOR, "EOR", ABSOLUTE, 3, 4},
	0x5D: opcode{EOR, "EOR", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x59: opcode{EOR, "EOR", ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0x41: opcode{EOR, "EOR", INDIRECT_X, 2, 6},
	0x51: opcode{EOR, "EOR", INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	0xE6: opcode{INC, "INC", ZERO_PAGE, 2, 5},
	0xF6: opcode{INC, "INC", ZERO_PAGE_X, 2, 6},
	0xEE: opcode{INC, "INC", ABSOLUTE, 3, 6},
	0xFE: opcode{INC, "INC", ABSOLUTE_X, 3, 7},
	0xE8: opcode{INX, "INX", IMPLICIT, 1, 2},
	0xC8: opcode{INY, "INY", IMPLICIT, 1, 2},
	0x4C: opcode{JMP, "JMP", ABSOLUTE, 3, 3},
	0x6C: opcode{JMP, "JMP", INDIRECT, 3, 5},
	0x20: opcode{JSR, "JSR", ABSOLUTE, 3, 6},
	0xA9: opcode{LDA, "LDA", IMMEDIATE, 2, 2},
	0xA5: opcode{LDA, "LDA", ZERO_PAGE, 2, 3},
	0xB5: opcode{LDA, "LDA", ZERO_PAGE_X, 2, 4},
	0xAD: opcode{LDA, "LDA", ABSOLUTE, 3, 4},
	0xBD: opcode{LDA, "LDA", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0xB9: opcode{LDA, "LDA", ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xA1: opcode{LDA, "LDA", INDIRECT_X, 2, 6},
	0xB1: opcode{LDA, "LDA", INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	0xA2: opcode{LDX, "LDX", IMMEDIATE, 2, 2},
	0xA6: opcode{LDX, "LDX", ZERO_PAGE, 2, 3},
	0xB6: opcode{LDX, "LDX", ZERO_PAGE_Y, 2, 4},
	0xAE: opcode{LDX, "LDX", ABSOLUTE, 3, 4},
	0xBE: opcode{LDX, "LDX", ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xA0: opcode{LDY, "LDY", IMMEDIATE, 2, 2},
	0xA4: opcode{LDY, "LDY", ZERO_PAGE, 2, 3},
	0xB4: opcode{LDY, "LDY", ZERO_PAGE_X, 2, 4},
	0xAC: opcode{LDY, "LDY", ABSOLUTE, 3, 4},
	0xBC: opcode{LDY, "LDY", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x4A: opcode{LSR, "LSR", ACCUMULATOR, 1, 2},
	0x46: opcode{LSR, "LSR", ZERO_PAGE, 2, 5},
	0x56: opcode{LSR, "LSR", ZERO_PAGE_X, 2, 6},
	0x4E: opcode{LSR, "LSR", ABSOLUTE, 3, 6},
	0x5E: opcode{LSR, "LSR", ABSOLUTE_X, 3, 7},
	0xEA: opcode{NOP, "NOP", IMPLICIT, 1, 2},
	0x09: opcode{ORA, "ORA", IMMEDIATE, 2, 2},
	0x05: opcode{ORA, "ORA", ZERO_PAGE, 2, 3},
	0x15: opcode{ORA, "ORA", ZERO_PAGE_X, 3, 4},
	0x0D: opcode{ORA, "ORA", ABSOLUTE, 3, 4},
	0x1D: opcode{ORA, "ORA", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x19: opcode{ORA, "ORA", ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0x01: opcode{ORA, "ORA", INDIRECT_X, 2, 6},
	0x11: opcode{ORA, "ORA", INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	0x48: opcode{PHA, "PHA", IMPLICIT, 1, 3},
	0x08: opcode{PHP, "PHP", IMPLICIT, 1, 3},
	0x68: opcode{PLA, "PLA", IMPLICIT, 1, 4},
	0x28: opcode{PLP, "PLP", IMPLICIT, 1, 4},
	0x2A: opcode{ROL, "ROL", ACCUMULATOR, 1, 2},
	0x26: opcode{ROL, "ROL", ZERO_PAGE, 2, 5},
	0x36: opcode{ROL, "ROL", ZERO_PAGE_X, 2, 6},
	0x2E: opcode{ROL, "ROL", ABSOLUTE, 3, 6},
	0x3E: opcode{ROL, "ROL", ABSOLUTE_X, 3, 7},
	0x6A: opcode{ROR, "ROR", ACCUMULATOR, 1, 2},
	0x66: opcode{ROR, "ROR", ZERO_PAGE, 2, 5},
	0x76: opcode{ROR, "ROR", ZERO_PAGE_X, 2, 6},
	0x6E: opcode{ROR, "ROR", ABSOLUTE, 3, 6},
	0x7E: opcode{ROR, "ROR", ABSOLUTE_X, 3, 7},
	0x40: opcode{RTI, "RTI", IMPLICIT, 1, 6},
	0x60: opcode{RTS, "RTS", IMPLICIT, 1, 6},
	0xE9: opcode{SBC, "SBC", IMMEDIATE, 2, 2},
	0xE5: opcode{SBC, "SBC", ZERO_PAGE, 2, 3},
	0xF5: opcode{SBC, "SBC", ZERO_PAGE_X, 2, 4},
	0xED: opcode{SBC, "SBC", ABSOLUTE, 3, 4},
	0xFD: opcode{SBC, "SBC", ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0xF9: opcode{SBC, "SBC", ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xE1: opcode{SBC, "SBC", INDIRECT_X, 2, 6},
	0xF1: opcode{SBC, "SBC", INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	0x38: opcode{SEC, "SEC", IMPLICIT, 1, 2},
	0xF8: opcode{SED, "SED", IMPLICIT, 1, 2},
	0x78: opcode{SEI, "SEI", IMPLICIT, 1, 2},
	0x85: opcode{STA, "STA", ZERO_PAGE, 2, 3},
	0x95: opcode{STA, "STA", ZERO_PAGE_X, 2, 4},
	0x8D: opcode{STA, "STA", ABSOLUTE, 3, 4},
	0x9D: opcode{STA, "STA", ABSOLUTE_X, 3, 5},
	0x99: opcode{STA, "STA", ABSOLUTE_Y, 3, 5},
	0x81: opcode{STA, "STA", INDIRECT_X, 2, 6},
	0x91: opcode{STA, "STA", INDIRECT_Y, 2, 6},
	0x86: opcode{STX, "STX", ZERO_PAGE, 2, 3},
	0x96: opcode{STX, "STX", ZERO_PAGE_Y, 2, 4},
	0x8E: opcode{STX, "STX", ABSOLUTE, 3, 4},
	0x84: opcode{STY, "STY", ZERO_PAGE, 2, 3},
	0x94: opcode{STY, "STY", ZERO_PAGE_X, 2, 4},
	0x8C: opcode{STY, "STY", ABSOLUTE, 3, 4},
	0xAA: opcode{TAX, "TAX", IMPLICIT, 1, 2},
	0xA8: opcode{TAY, "TAY", IMPLICIT, 1, 2},
	0xBA: opcode{TSX, "TSX", IMPLICIT, 1, 2},
	0x8A: opcode{TXA, "TXA", IMPLICIT, 1, 2},
	0x9A: opcode{TXS, "TXS", IMPLICIT, 1, 2},
	0x98: opcode{TYA, "TYA", IMPLICIT, 1, 2},
}

// How much addressable memory we have
const MEM_SIZE = math.MaxUint16 + 1

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

// type cpu implements all of the machine state for the 6502
type cpu struct {
	acc    uint8   // main register
	x, y   uint8   // index registers
	status uint8   // a register for storing various status bits
	sp     uint8   // stack pointer - stack is 0x0100-0x01FF so only 8 bits needed
	pc     uint16  // the program counter
	mem    *memory // 64k addressable memory
	cycles uint8   // how many cycles to wait until next instruction
}

func (c *cpu) String() string {
	return fmt.Sprintf("A,X,Y: %4d, %4d, %4d; PC: 0x%04x, SP: 0x%02x, P: %s; OP: %s", c.acc, c.x, c.y, c.pc, c.sp, statusString(c.status), opcodes[c.mem.read(c.pc)])
}

func New(m mappers.Mapper) *cpu {
	// Power on state values from:
	// https://nesdev-wiki.nes.science/wikipages/CPU_ALL.xhtml#Power_up_state
	// B is not normally visible in the register, but per docs, is
	// set at startup.
	c := &cpu{
		sp:     0xFD,
		mem:    newMemory(RAM_SIZE, m),
		status: UNUSED_STATUS_FLAG | STATUS_FLAG_BREAK | STATUS_FLAG_INTERRUPT_DISABLE,
	}
	c.pc = c.memRead16(INT_RESET)
	return c
}

var invalidInstruction = errors.New("invalid instruction")

func (c *cpu) getInst() (opcode, error) {
	m := c.memRead(c.pc)
	op, ok := opcodes[m]
	if !ok {
		return opcodes[0x00], fmt.Errorf("pc: %d, inst: 0x%02x - %w", c.pc, m, invalidInstruction)
	}

	return op, nil
}

// memRead returns the byte from memory at addr
func (c *cpu) memRead(addr uint16) uint8 {
	return c.mem.read(addr)
}

// memRange returns a slice of memory addresses from low to
// high. Mostly useful for debugging.
func (c *cpu) memRange(low, high uint16) []uint8 {
	ret := make([]uint8, high-low)
	for i := low; i <= high; i += 1 {
		ret = append(ret, c.mem.read(uint16(i)))
	}

	return ret
}

// memWrite writes val to memory at addr
func (c *cpu) memWrite(addr uint16, val uint8) {
	c.mem.write(addr, val)
}

// memRead16 returns the two bytes from memory at addr (lower byte is
// first).
func (c *cpu) memRead16(addr uint16) uint16 {
	lsb := uint16(c.memRead(addr))
	msb := uint16(c.memRead(addr + 1))

	return (msb << 8) | lsb
}

func (c *cpu) memWrite16(addr, val uint16) {
	c.memWrite(addr, uint8(val&0x00FF))
	c.memWrite(addr+1, uint8(val>>8))
}

// getOperandAddr takes a mode and returns an address for the operand
// referenced by the program counter. It assumes that the counter was
// incremented past the actual instruction itself.
func (c *cpu) getOperandAddr(mode uint8) uint16 {
	var addr uint16
	switch mode {
	case ACCUMULATOR:
		panic("ACCUMULATOR Address mode should never use this method")
	case IMPLICIT:
		panic("IMPLICIT Address mode should never use this method")
	case IMMEDIATE:
		addr = c.pc
	case ZERO_PAGE:
		addr = uint16(c.memRead(c.pc))
	case ZERO_PAGE_X:
		return uint16(c.memRead(c.pc) + c.x)
	case ZERO_PAGE_Y:
		return uint16(c.memRead(c.pc) + c.y)
	case ABSOLUTE:
		return c.memRead16(c.pc)
	case ABSOLUTE_X:
		a := c.memRead16(c.pc)
		addr = a + uint16(c.x)
		c.cycles += extraCycles(a, addr)
	case ABSOLUTE_Y:
		a := c.memRead16(c.pc)
		addr = a + uint16(c.y)
		c.cycles += extraCycles(a, addr)
	case INDIRECT:
		return c.memRead16(c.memRead16(c.pc))
	case INDIRECT_X:
		return c.memRead16(uint16(c.memRead(c.pc) + c.x))
	case INDIRECT_Y:
		a := c.memRead16(uint16(c.memRead(c.pc)))
		addr = a + uint16(c.y)
		c.cycles += extraCycles(a, addr)
	case RELATIVE:
		// Relative from PC at time of instruction
		// execution. We advance pc as soon as we eat the byte
		// from memory to decode the instruction, so we need
		// to account for that here and step over the relative
		// argument while calculating the new target address.
		addr = (c.pc + 1) + uint16(int8(c.memRead(c.pc)))
	default:
		panic("Invalid addressing mode")

	}

	return addr
}

func (c *cpu) reset() {
	// Reset is the only time we should ever touch the unused flag
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE | UNUSED_STATUS_FLAG)
	c.pc = c.memRead16(INT_RESET)
}

func readAddress(prompt string) uint16 {
	var a uint16
	fmt.Printf(prompt)
	fmt.Scanf("%04x\n", &a)
	return a
}

func (c *cpu) BIOS(ctx context.Context) {

	sigQuit := make(chan os.Signal, 1)
	signal.Notify(sigQuit, syscall.SIGINT, syscall.SIGTERM)

	breaks := make(map[uint16]struct{})

	for {
		fmt.Printf("%s\n\n", c)
		fmt.Println("(B)reak - add breakpoint")
		fmt.Println("(C)lear - cleear breakpoints")
		fmt.Println("(R)un - run to completion")
		fmt.Println("(S)step - step the cpu one instruction")
		fmt.Println("R(e)set - hit the reset button")
		fmt.Println("(M)memory - select a memory range to display")
		fmt.Println("S(t)ack - show last 3 items on the stack")
		fmt.Println("(I)instruction - show instruction memory locations")
		fmt.Println("(Q)uit - shutdown the gintentdo")
		fmt.Printf("Choice: ")

		var in rune
		fmt.Scanf("%c\n", &in)

		switch in {
		case 'b', 'B':
			breaks[readAddress("Breakpoint (eg: ff15): ")] = struct{}{}
		case 'c', 'C':
			breaks = make(map[uint16]struct{})
		case 'q', 'Q':
			return
		case 'r', 'R':
			cctx, cancel := context.WithCancel(ctx)
			go func(ctx context.Context) {
				for {
					select {
					case <-sigQuit:
						cancel()
					case <-ctx.Done():
						return
					}
				}
			}(cctx)
			c.Run(cctx, breaks)
		case 's', 'S':
			c.step()
		case 't', 'T':
			fmt.Println()
			i := 0
			for {
				m := c.getStackAddr() + uint16(i)
				fmt.Printf("0x%04x: 0x%02x ", m, c.memRead(m))
				if m == 0x00ff || i == 2 {
					break
				}
				i += 1
			}
			fmt.Printf("\n\n")
		case 'i', 'I':
			fmt.Println()
			op := opcodes[c.memRead(c.pc)]
			for i := 0; i < int(op.bytes); i++ {
				m := c.pc + uint16(i)
				fmt.Printf("0x%04x: 0x%02x ", m, c.memRead(m))
			}
			fmt.Printf("\n\n")
		case 'e', 'E':
			c.reset()
		case 'm', 'M':
			fmt.Println()
			low := readAddress("Low address (eg f00d): ")
			high := readAddress("High address (eg beef): ")
			fmt.Println()

			x := 1
			i := low
			for {
				fmt.Printf("0x%04x: 0x%02x ", i, c.memRead(i))
				if x%5 == 0 {
					fmt.Println()
				}
				if i == high || i == math.MaxUint16 {
					break
				}
				x += 1
				i += 1
			}
			fmt.Printf("\n\n")
		}
	}
}

func (c *cpu) Run(ctx context.Context, breaks map[uint16]struct{}) {
	// https://www.nesdev.org/wiki/CPU#Frequencies
	t := time.NewTicker(time.Nanosecond * 559)
	for {
		select {
		case <-t.C:
			c.step()
			fmt.Println(c)
		case <-ctx.Done():
			return
		}

		if _, ok := breaks[c.pc]; ok {
			fmt.Printf("Hit breakpoint at 0%04x\n", c.pc)
			return
		}
	}
}

func (c *cpu) step() {
	if c.cycles > 0 {
		c.cycles -= 1
		return
	}

	op, err := c.getInst()
	if err != nil {
		panic(err)
	}

	c.cycles += op.cycles
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
}

// setNegativeAndZeroFlags sets the STATUS_FLAG_NEGATIVE and
// STATUS_FLAG_ZERO bits of the status register accordingly for the
// value specified in n.
func (c *cpu) setNegativeAndZeroFlags(n uint8) {
	if n == 0 {
		c.flagsOn(STATUS_FLAG_ZERO)
	} else {
		c.flagsOff(STATUS_FLAG_ZERO)
	}

	if n&0b1000_0000 != 0 {
		c.flagsOn(STATUS_FLAG_NEGATIVE)
	} else {
		c.flagsOff(STATUS_FLAG_NEGATIVE)
	}
}

func (c *cpu) getStackAddr() uint16 {
	return STACK_PAGE + uint16(c.sp)
}

func (c *cpu) pushStack(val uint8) {
	c.memWrite(c.getStackAddr(), val)
	c.sp -= 1
}

func (c *cpu) popStack() uint8 {
	c.sp += 1
	return c.memRead(c.getStackAddr())
}

func (c *cpu) pushAddress(addr uint16) {
	c.pushStack(uint8(addr >> 8))     // high
	c.pushStack(uint8(addr & 0x00FF)) // low
}

func (c *cpu) popAddress() uint16 {
	return uint16(c.popStack()) | (uint16(c.popStack()) << 8)
}

// flagsOn forces the flags in mask (STATUS_FLAG_XXX|STATUS_FLAG_YYY)
// on in the status register.
func (c *cpu) flagsOn(mask uint8) {
	c.status = c.status | mask
}

// flagsOff forces the flags in mask (STATUS_FLAG_XXX|STATUS_FLAG_YYY)
// off in the status register.
func (c *cpu) flagsOff(mask uint8) {
	c.status = c.status &^ mask
}

// extraCycles returns 0 if addr1 and add2 are in the same page, 1
// otherwise. This is useful for instructions that take a variable
// number of cycles, depending on whether or not a page boundary is
// crossed.
func extraCycles(addr1, addr2 uint16) uint8 {
	if addr1&0xFF00 != addr2&0xFF00 {
		return 1
	}
	return 0
}

// branch will adjust the PC conditionally based on whether the mask
// bits are set and the resulting comparison is expected to be true or
// false. This allows you to check for STATUS_FLAG being set or
// cleared by: branch(STATUS_FLAG_OVERFLOW, RELATIVE, false) -> branch
// when OVERFLOW not set.
func (c *cpu) branch(mask uint8, predicate bool) {
	if (c.status&mask > 0) == predicate {
		a := c.getOperandAddr(RELATIVE)
		// Branching instructions take an extra cycle if they
		// cause a page break pc-1 because we increment it
		// right after reading the op, but that's where we
		// branch from so that's where we compare for page
		// break
		c.cycles += extraCycles(a, c.pc-1)
		c.cycles += 1 // successful branches take an extra cycle
		c.pc = a
	}
}

// addWithOverflow adds b to c.acc handling overflow, carry and ZN
// flag setting as appropriate.
func (c *cpu) addWithOverflow(b uint8) {
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

// baseCMP does comparison operations on a and b, setting flags
// accordingly.
func (c *cpu) baseCMP(a, b uint8) {
	c.setNegativeAndZeroFlags(a - b)
	if a >= b {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) ADC(mode uint8) {
	c.addWithOverflow(c.memRead(c.getOperandAddr(mode)))
}

func (c *cpu) AND(mode uint8) {
	c.acc = c.acc & c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) ASL(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = c.acc << 1
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.memRead(addr)
		nv = ov << 1
		c.memWrite(addr, nv)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&0x80 != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) BCC(mode uint8) {
	c.branch(STATUS_FLAG_CARRY, false)
}

func (c *cpu) BCS(mode uint8) {
	c.branch(STATUS_FLAG_CARRY, true)
}

func (c *cpu) BEQ(mode uint8) {
	c.branch(STATUS_FLAG_ZERO, true)
}

func (c *cpu) BIT(mode uint8) {
	o := c.memRead(c.getOperandAddr(mode))

	c.flagsOff(STATUS_FLAG_NEGATIVE | STATUS_FLAG_OVERFLOW | STATUS_FLAG_ZERO)
	var flags uint8
	if (o & c.acc) == 0 {
		flags = flags | STATUS_FLAG_ZERO
	}
	flags = flags | (o & (STATUS_FLAG_NEGATIVE | STATUS_FLAG_OVERFLOW))

	c.flagsOn(flags)
}

func (c *cpu) BMI(mode uint8) {
	c.branch(STATUS_FLAG_NEGATIVE, true)
}

func (c *cpu) BNE(mode uint8) {
	c.branch(STATUS_FLAG_ZERO, false)
}

func (c *cpu) BPL(mode uint8) {
	c.branch(STATUS_FLAG_NEGATIVE, false)
}

func (c *cpu) BRK(mode uint8) {
	// BRK is 2 bytes
	c.pushAddress(c.pc + 1)
	c.pushStack(c.status | STATUS_FLAG_BREAK)
	c.pc = c.memRead16(INT_BRK)
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) BVC(mode uint8) {
	c.branch(STATUS_FLAG_OVERFLOW, false)
}

func (c *cpu) BVS(mode uint8) {
	c.branch(STATUS_FLAG_OVERFLOW, true)
}

func (c *cpu) CLC(mode uint8) {
	c.flagsOff(STATUS_FLAG_CARRY)
}

func (c *cpu) CLD(mode uint8) {
	c.flagsOff(STATUS_FLAG_DECIMAL)
}

func (c *cpu) CLI(mode uint8) {
	c.flagsOff(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) CLV(mode uint8) {
	c.flagsOff(STATUS_FLAG_OVERFLOW)
}

func (c *cpu) CMP(mode uint8) {
	c.baseCMP(c.acc, c.memRead(c.getOperandAddr(mode)))
}

func (c *cpu) CPX(mode uint8) {
	c.baseCMP(c.x, c.memRead(c.getOperandAddr(mode)))
}

func (c *cpu) CPY(mode uint8) {
	c.baseCMP(c.y, c.memRead(c.getOperandAddr(mode)))
}

func (c *cpu) DEC(mode uint8) {
	a := c.getOperandAddr(mode)
	c.memWrite(a, c.memRead(a)-1)
	c.setNegativeAndZeroFlags(c.memRead(a))
}

func (c *cpu) DEX(mode uint8) {
	c.x -= 1
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) DEY(mode uint8) {
	c.y -= 1
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) EOR(mode uint8) {
	c.acc = c.acc ^ c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) INC(mode uint8) {
	a := c.getOperandAddr(mode)
	c.memWrite(a, c.memRead(a)+1)
	c.setNegativeAndZeroFlags(c.memRead(a))
}

func (c *cpu) INX(mode uint8) {
	c.x += 1
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) INY(mode uint8) {
	c.y += 1
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) JMP(mode uint8) {
	c.pc = c.getOperandAddr(mode)
}

func (c *cpu) JSR(mode uint8) {
	c.pushAddress(c.pc + 1) // this is the second byte of the JSR argument
	c.pc = c.getOperandAddr(mode)
}

func (c *cpu) LDA(mode uint8) {
	c.acc = c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) LDX(mode uint8) {
	c.x = c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) LDY(mode uint8) {
	c.y = c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) LSR(mode uint8) {
	var ov, nv uint8
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = c.acc >> 1
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.memRead(addr)
		nv = ov >> 1
		c.memWrite(addr, nv)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&STATUS_FLAG_CARRY != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}

}

func (c *cpu) NOP(mode uint8) {
	return
}

func (c *cpu) ORA(mode uint8) {
	c.acc = c.acc | c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) PHA(mode uint8) {
	c.pushStack(c.acc)
}

func (c *cpu) PHP(mode uint8) {
	// 6502 always sets BREAK when pushing the status register to
	// the stack
	c.pushStack(c.status | STATUS_FLAG_BREAK)
}

func (c *cpu) PLA(mode uint8) {
	c.acc = c.popStack()
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) PLP(mode uint8) {
	c.status = c.popStack() & ^uint8(STATUS_FLAG_BREAK)
}

func (c *cpu) ROL(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = bits.RotateLeft8(ov, 1) | (c.status & STATUS_FLAG_CARRY)
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.memRead(addr)
		c.memWrite(addr, bits.RotateLeft8(ov, 1)|(c.status&STATUS_FLAG_CARRY))
		nv = c.memRead(addr)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&0x80 != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) ROR(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = bits.RotateLeft8(ov, -1) | ((c.status & STATUS_FLAG_CARRY) << 7)
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.memRead(addr)
		c.memWrite(addr, bits.RotateLeft8(ov, -1)|((c.status&STATUS_FLAG_CARRY)<<7))
		nv = c.memRead(addr)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&STATUS_FLAG_CARRY != 0 { // was carry bit set in the old _value_?
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) RTI(mode uint8) {
	c.status = c.popStack()
	c.pc = c.popAddress()
}

func (c *cpu) RTS(mode uint8) {
	c.pc = c.popAddress() + 1
}

func (c *cpu) SBC(mode uint8) {
	c.addWithOverflow(^c.memRead(c.getOperandAddr(mode)))
}

func (c *cpu) SEC(mode uint8) {
	c.flagsOn(STATUS_FLAG_CARRY)
}

func (c *cpu) SED(mode uint8) {
	c.flagsOn(STATUS_FLAG_DECIMAL)
}

func (c *cpu) SEI(mode uint8) {
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) STA(mode uint8) {
	c.memWrite(c.getOperandAddr(mode), c.acc)
}

func (c *cpu) STX(mode uint8) {
	c.memWrite(c.getOperandAddr(mode), c.x)
}

func (c *cpu) STY(mode uint8) {
	c.memWrite(c.getOperandAddr(mode), c.y)
}

func (c *cpu) TAX(mode uint8) {
	c.x = c.acc
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) TAY(mode uint8) {
	c.y = c.acc
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) TSX(mode uint8) {
	c.x = c.sp
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) TXA(mode uint8) {
	c.acc = c.x
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) TXS(mode uint8) {
	c.sp = c.x
}

func (c *cpu) TYA(mode uint8) {
	c.acc = c.y
	c.setNegativeAndZeroFlags(c.acc)
}

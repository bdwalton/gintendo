// Package mos6502 implements the MOS Technologies 6502 processor
// https://en.wikipedia.org/wiki/MOS_Technology_6502
package mos6502

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

// 6502 Processor Status Flags
// https://www.nesdev.org/obelisk-6502-guide/registers.html
const (
	STATUS_FLAG_CARRY             = 1 << 0 // C
	STATUS_FLAG_ZERO              = 1 << 1 // Z
	STATUS_FLAG_INTERRUPT_DISABLE = 1 << 2 // I
	STATUS_FLAG_DECIMAL           = 1 << 3 // D
	STATUS_FLAG_BREAK             = 1 << 4 // B
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

var opnames map[uint8]string = map[uint8]string{ADC: "ADC", AND: "AND", ASL: "ASL", BCC: "BCC", BCS: "BCS", BEQ: "BEQ", BIT: "BIT", BMI: "BMI", BNE: "BNE", BPL: "BPL", BRK: "BRK", BVC: "BVC", BVS: "BVS", CLC: "CLC", CLD: "CLD", CLI: "CLI", CLV: "CLV", CMP: "CMP", CPX: "CPX", CPY: "CPY", DEC: "DEC", DEX: "DEX", DEY: "DEY", EOR: "EOR", INC: "INC", INX: "INX", INY: "INY", JMP: "JMP", JSR: "JSR", LDA: "LDA", LDX: "LDX", LDY: "LDY", LSR: "LSR", NOP: "NOP", ORA: "ORA", PHA: "PHA", PHP: "PHP", PLA: "PLA", PLP: "PLP", ROL: "ROL", ROR: "ROR", RTI: "RTI", RTS: "RTS", SBC: "SBC", SEC: "SEC", SED: "SED", SEI: "SEI", STA: "STA", STX: "STX", STY: "STY", TAX: "TAX", TAY: "TAY", TSX: "TSX", TXA: "TXA", TXS: "TXS", TYA: "TYA"}

type opcode struct {
	inst    uint8 // The instruction
	mode    uint8 // The memory addressing mode to use
	bytes   uint8 // The number of bytes consumed by operands
	_cycles uint8 // The number of cycles consumed by the instruction
}

func (o opcode) String() string {
	return fmt.Sprintf("{%s, %s}", opnames[o.inst], modenames[o.mode])
}

var opcodes map[uint8]opcode = map[uint8]opcode{
	// ADC
	0x69: opcode{ADC, IMMEDIATE, 2, 2},
	0x65: opcode{ADC, ZERO_PAGE, 2, 3},
	0x75: opcode{ADC, ZERO_PAGE_X, 2, 4},
	0x6D: opcode{ADC, ABSOLUTE, 3, 4},
	0x7D: opcode{ADC, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x79: opcode{ADC, ABSOLUTE_Y, 3, 4 /* +1 if page crossed*/},
	0x61: opcode{ADC, INDIRECT_X, 2, 6},
	0x71: opcode{ADC, INDIRECT_Y, 2, 5 /* +1 if page crossed*/},
	// AND
	0x29: opcode{AND, IMMEDIATE, 2, 2},
	0x25: opcode{AND, ZERO_PAGE, 2, 3},
	0x35: opcode{AND, ZERO_PAGE_X, 2, 4},
	0x2D: opcode{AND, ABSOLUTE, 3, 4},
	0x3D: opcode{AND, ABSOLUTE_X, 3, 4 /* + 1 if page crossed*/},
	0x39: opcode{AND, ABSOLUTE_Y, 3, 4 /* +1 if page crossed*/},
	0x21: opcode{AND, INDIRECT_X, 2, 6},
	0x31: opcode{AND, INDIRECT_Y, 2, 5 /* +1 if page crossed*/},
	// ASL
	0x0A: opcode{ASL, ACCUMULATOR, 1, 2},
	0x06: opcode{ASL, ZERO_PAGE, 2, 5},
	0x16: opcode{ASL, ZERO_PAGE_X, 2, 6},
	0x0E: opcode{ASL, ABSOLUTE, 3, 6},
	0x1E: opcode{ASL, ABSOLUTE_X, 3, 7},
	// BCC
	0x90: opcode{BCC, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BCS
	0xB0: opcode{BCS, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BEQ
	0xF0: opcode{BEQ, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BIT
	0x24: opcode{BIT, ZERO_PAGE, 2, 3},
	0x2C: opcode{BIT, ABSOLUTE, 3, 4},
	// BMI
	0x30: opcode{BMI, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BNE
	0xD0: opcode{BNE, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BPL
	0x10: opcode{BPL, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BRK
	0x00: opcode{BRK, IMPLICIT, 1, 7},
	// BVC
	0x50: opcode{BVC, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// BVS
	0x70: opcode{BVS, RELATIVE, 2, 2 /* +1 if branch successes +2 if to a new page */},
	// CLC
	0x18: opcode{CLC, IMPLICIT, 1, 2},
	// CLD
	0xD8: opcode{CLD, IMPLICIT, 1, 2},
	// CLI
	0x58: opcode{CLI, IMPLICIT, 1, 2},
	// CLV
	0xB8: opcode{CLV, IMPLICIT, 1, 2},
	// CMP
	0xC9: opcode{CMP, IMMEDIATE, 2, 2},
	0xC5: opcode{CMP, ZERO_PAGE, 2, 3},
	0xD5: opcode{CMP, ZERO_PAGE_X, 2, 4},
	0xCD: opcode{CMP, ABSOLUTE, 3, 4},
	0xDD: opcode{CMP, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0xD9: opcode{CMP, ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xC1: opcode{CMP, INDIRECT_X, 2, 6},
	0xD1: opcode{CMP, INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	// CPX
	0xE0: opcode{CPX, IMMEDIATE, 2, 2},
	0xE4: opcode{CPX, ZERO_PAGE, 2, 3},
	0xEC: opcode{CPX, ABSOLUTE, 3, 4},
	// CPY
	0xC0: opcode{CPY, IMMEDIATE, 2, 2},
	0xC4: opcode{CPY, ZERO_PAGE, 2, 3},
	0xCC: opcode{CPY, ABSOLUTE, 3, 4},
	// DEC
	0xC6: opcode{DEC, ZERO_PAGE, 2, 5},
	0xD6: opcode{DEC, ZERO_PAGE_X, 2, 6},
	0xCE: opcode{DEC, ABSOLUTE, 3, 6},
	0xDE: opcode{DEC, ABSOLUTE_X, 3, 7},
	// DEX
	0xCA: opcode{DEX, IMPLICIT, 1, 2},
	// DEY
	0x88: opcode{DEY, IMPLICIT, 1, 2},
	// EOR
	0x49: opcode{EOR, IMMEDIATE, 2, 2},
	0x45: opcode{EOR, ZERO_PAGE, 2, 3},
	0x55: opcode{EOR, ZERO_PAGE_X, 2, 4},
	0x4D: opcode{EOR, ABSOLUTE, 3, 4},
	0x5D: opcode{EOR, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x59: opcode{EOR, ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0x41: opcode{EOR, INDIRECT_X, 2, 6},
	0x51: opcode{EOR, INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	// INC
	0xE6: opcode{INC, ZERO_PAGE, 2, 5},
	0xF6: opcode{INC, ZERO_PAGE_X, 2, 6},
	0xEE: opcode{INC, ABSOLUTE, 3, 6},
	0xFE: opcode{INC, ABSOLUTE_X, 3, 7},
	// INX
	0xE8: opcode{INX, IMPLICIT, 1, 2},
	// INY
	0xC8: opcode{INY, IMPLICIT, 1, 2},
	// JMP
	0x4C: opcode{JMP, ABSOLUTE, 3, 3},
	0x6C: opcode{JMP, INDIRECT, 3, 5},
	// JSR
	0x20: opcode{JSR, ABSOLUTE, 3, 6},
	// LDA
	0xA9: opcode{LDA, IMMEDIATE, 2, 2},
	0xA5: opcode{LDA, ZERO_PAGE, 2, 3},
	0xB5: opcode{LDA, ZERO_PAGE_X, 2, 4},
	0xAD: opcode{LDA, ABSOLUTE, 3, 4},
	0xBD: opcode{LDA, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0xB9: opcode{LDA, ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xA1: opcode{LDA, INDIRECT_X, 2, 6},
	0xB1: opcode{LDA, INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	// LDX
	0xA2: opcode{LDX, IMMEDIATE, 2, 2},
	0xA6: opcode{LDX, ZERO_PAGE, 2, 3},
	0xB6: opcode{LDX, ZERO_PAGE_Y, 2, 4},
	0xAE: opcode{LDX, ABSOLUTE, 3, 4},
	0xBE: opcode{LDX, ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	// LDY
	0xA0: opcode{LDY, IMMEDIATE, 2, 2},
	0xA4: opcode{LDY, ZERO_PAGE, 2, 3},
	0xB4: opcode{LDY, ZERO_PAGE_X, 2, 4},
	0xAC: opcode{LDY, ABSOLUTE, 3, 4},
	0xBC: opcode{LDY, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	// LSR
	0x4A: opcode{LSR, ACCUMULATOR, 1, 2},
	0x46: opcode{LSR, ZERO_PAGE, 2, 5},
	0x56: opcode{LSR, ZERO_PAGE_X, 2, 6},
	0x4E: opcode{LSR, ABSOLUTE, 3, 6},
	0x5E: opcode{LSR, ABSOLUTE_X, 3, 7},
	// NOP
	0xEA: opcode{NOP, IMPLICIT, 1, 2},
	// ORA
	0x09: opcode{ORA, IMMEDIATE, 2, 2},
	0x05: opcode{ORA, ZERO_PAGE, 2, 3},
	0x15: opcode{ORA, ZERO_PAGE_X, 3, 4},
	0x0D: opcode{ORA, ABSOLUTE, 3, 4},
	0x1D: opcode{ORA, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0x19: opcode{ORA, ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0x01: opcode{ORA, INDIRECT_X, 2, 6},
	0x11: opcode{ORA, INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	// PHA
	0x48: opcode{PHA, IMPLICIT, 1, 3},
	// PHP
	0x08: opcode{PHP, IMPLICIT, 1, 3},
	// PLA
	0x68: opcode{PLA, IMPLICIT, 1, 4},
	// PLP
	0x28: opcode{PLP, IMPLICIT, 1, 4},
	// ROL
	0x2A: opcode{ROL, ACCUMULATOR, 1, 2},
	0x26: opcode{ROL, ZERO_PAGE, 2, 5},
	0x36: opcode{ROL, ZERO_PAGE_X, 2, 6},
	0x2E: opcode{ROL, ABSOLUTE, 3, 6},
	0x3E: opcode{ROL, ABSOLUTE_X, 3, 7},
	// ROR
	0x6A: opcode{ROR, ACCUMULATOR, 1, 2},
	0x66: opcode{ROR, ZERO_PAGE, 2, 5},
	0x76: opcode{ROR, ZERO_PAGE_X, 2, 6},
	0x6E: opcode{ROR, ABSOLUTE, 3, 6},
	0x7E: opcode{ROR, ABSOLUTE_X, 3, 7},
	// RTI
	0x40: opcode{RTI, IMPLICIT, 1, 6},
	// RTS
	0x60: opcode{RTS, IMPLICIT, 1, 6},
	// SBC
	0xE9: opcode{SBC, IMMEDIATE, 2, 2},
	0xE5: opcode{SBC, ZERO_PAGE, 2, 3},
	0xF5: opcode{SBC, ZERO_PAGE_X, 2, 4},
	0xED: opcode{SBC, ABSOLUTE, 3, 4},
	0xFD: opcode{SBC, ABSOLUTE_X, 3, 4 /* +1 if page crossed */},
	0xF9: opcode{SBC, ABSOLUTE_Y, 3, 4 /* +1 if page crossed */},
	0xE1: opcode{SBC, INDIRECT_X, 2, 6},
	0xF1: opcode{SBC, INDIRECT_Y, 2, 5 /* +1 if page crossed */},
	// SEC
	0x38: opcode{SEC, IMPLICIT, 1, 2},
	// SED
	0xF8: opcode{SED, IMPLICIT, 1, 2},
	// SEI
	0x78: opcode{SEI, IMPLICIT, 1, 2},
	// STA
	0x85: opcode{STA, ZERO_PAGE, 2, 3},
	0x95: opcode{STA, ZERO_PAGE_X, 2, 4},
	0x8D: opcode{STA, ABSOLUTE, 3, 4},
	0x9D: opcode{STA, ABSOLUTE_X, 3, 5},
	0x99: opcode{STA, ABSOLUTE_Y, 3, 5},
	0x81: opcode{STA, INDIRECT_X, 2, 6},
	0x91: opcode{STA, INDIRECT_Y, 2, 6},
	// STX
	0x86: opcode{STX, ZERO_PAGE, 2, 3},
	0x96: opcode{STX, ZERO_PAGE_Y, 2, 4},
	0x8E: opcode{STX, ABSOLUTE, 3, 4},
	// STY
	0x84: opcode{STY, ZERO_PAGE, 2, 3},
	0x94: opcode{STY, ZERO_PAGE_X, 2, 4},
	0x8C: opcode{STY, ABSOLUTE, 3, 4},
	// TAX
	0xAA: opcode{TAX, IMPLICIT, 1, 2},
	// TAY
	0xA8: opcode{TAY, IMPLICIT, 1, 2},
	// TSX
	0xBA: opcode{TSX, IMPLICIT, 1, 2},
	// TXA
	0x8A: opcode{TXA, IMPLICIT, 1, 2},
	// TXS:
	0x9A: opcode{TXS, IMPLICIT, 1, 2},
	// TYA
	0x98: opcode{TYA, IMPLICIT, 1, 2},
}

// How much addressable memory we have
const MEM_SIZE = math.MaxUint16

// type cpu implements all of the machine state for the 6502
type cpu struct {
	acc    uint8  // main register
	x, y   uint8  // index registers
	status uint8  // a register for storing various status bits
	sp     uint8  // stack pointer - stack is 0x0100-0x01FF so only 8 bits needed
	pc     uint16 // the program counter
	memory [MEM_SIZE]uint8
}

func (c *cpu) String() string {
	return fmt.Sprintf("ACC: %d, X: %d, Y: %d, SP: %d, PC: %d, Inst: %s", c.acc, c.x, c.y, c.sp, c.pc, opcodes[c.memory[c.pc]])
}

func New() *cpu {
	return &cpu{sp: 0xFF}
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
	return c.memory[addr]
}

// memRange returns a slice of memory addresses from low to
// high. Mostly useful for debugging.
func (c *cpu) memRange(low, high uint16) []uint8 {
	return c.memory[low : high+1]
}

// writeMem writes val to memory at addr
func (c *cpu) writeMem(addr uint16, val uint8) {
	c.memory[addr] = val
}

// memRead16 returns the two bytes from memory at addr (lower byte is
// first).
func (c *cpu) memRead16(addr uint16) uint16 {
	lsb := uint16(c.memRead(addr))
	msb := uint16(c.memRead(addr + 1))

	return (msb << 8) | lsb
}

func (c *cpu) writeMem16(addr, val uint16) {
	c.writeMem(addr, uint8(val&0x00FF))
	c.writeMem(addr+1, uint8(val>>8))
}

// getOperandAddr takes a mode and returns an address for the operand
// referenced by the program counter. It assumes that the counter was
// incremented past the actual instruction itself.
func (c *cpu) getOperandAddr(mode uint8) uint16 {
	switch mode {
	case ACCUMULATOR:
		panic("ACCUMULATOR Address mode should never use this method")
	case IMPLICIT:
		panic("IMPLICIT Address mode should never use this method")
	case IMMEDIATE:
		return c.pc
	case ZERO_PAGE:
		return uint16(c.memRead(c.pc))
	case ZERO_PAGE_X:
		return uint16(c.memRead(c.pc) + c.x)
	case ZERO_PAGE_Y:
		return uint16(c.memRead(c.pc) + c.y)
	case ABSOLUTE:
		return c.memRead16(c.pc)
	case ABSOLUTE_X:
		return c.memRead16(c.pc) + uint16(c.x)
	case ABSOLUTE_Y:
		return c.memRead16(c.pc) + uint16(c.y)
	case INDIRECT:
		return c.memRead16(c.memRead16(c.pc))
	case INDIRECT_X:
		return c.memRead16(uint16(c.memRead(c.pc) + c.x))
	case INDIRECT_Y:
		return c.memRead16(uint16(c.memRead(c.pc))) + uint16(c.y)
	case RELATIVE:
		return c.pc + uint16(int16(int8(c.memRead(c.pc))))
	default:
		panic("Invalid addressing mode")

	}
}

func (c *cpu) step() {
	op, err := c.getInst()

	if err != nil {
		panic(err)
	}

	c.pc += 1

	switch op.inst {
	case AND:
		c.opAND(op.mode)
	case ASL:
		c.opASL(op.mode)
	case BEQ:
		c.opBEQ(op.mode)
	case BIT:
		c.opBIT(op.mode)
	case BMI:
		c.opBMI(op.mode)
	case BNE:
		c.opBNE(op.mode)
	case BPL:
		c.opBPL(op.mode)
	case BVC:
		c.opBVC(op.mode)
	case BVS:
		c.opBVS(op.mode)
	case CLC:
		c.opCLC(op.mode)
	case CLD:
		c.opCLD(op.mode)
	case CLI:
		c.opCLI(op.mode)
	case CLV:
		c.opCLV(op.mode)
	case DEC:
		c.opDEC(op.mode)
	case DEX:
		c.opDEX(op.mode)
	case DEY:
		c.opDEY(op.mode)
	case EOR:
		c.opEOR(op.mode)
	case INC:
		c.opINC(op.mode)
	case INX:
		c.opINX(op.mode)
	case INY:
		c.opINY(op.mode)
	case JMP:
		c.opJMP(op.mode)
	case JSR:
		c.opJSR(op.mode)
	case LDA:
		c.opLDA(op.mode)
	case LDX:
		c.opLDX(op.mode)
	case LDY:
		c.opLDY(op.mode)
	case LSR:
		c.opLSR(op.mode)
	case NOP:
		c.opNOP(op.mode)
	case ORA:
		c.opORA(op.mode)
	case PHA:
		c.opPHA(op.mode)
	case PHP:
		c.opPHP(op.mode)
	case PLA:
		c.opPLA(op.mode)
	case PLP:
		c.opPLP(op.mode)
	case ROL:
		c.opROL(op.mode)
	case ROR:
		c.opROR(op.mode)
	case RTS:
		c.opRTS(op.mode)
	case SEC:
		c.opSEC(op.mode)
	case SED:
		c.opSED(op.mode)
	case SEI:
		c.opSEI(op.mode)
	case STA:
		c.opSTA(op.mode)
	case STX:
		c.opSTX(op.mode)
	case STY:
		c.opSTY(op.mode)
	case TAX:
		c.opTAX(op.mode)
	case TAY:
		c.opTAY(op.mode)
	case TSX:
		c.opTSX(op.mode)
	case TXA:
		c.opTXA(op.mode)
	case TXS:
		c.opTXS(op.mode)
	case TYA:
		c.opTYA(op.mode)
	default:
		panic(fmt.Errorf("unimplemented instruction %s", op))
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
	c.writeMem(c.getStackAddr(), val)
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

// branch will adjust the PC conditionally based on whether the mask
// bits are set and the resulting comparison is expected to be true or
// false. This allows you to check for STATUS_FLAG being set or
// cleared by: branch(STATUS_FLAG_OVERFLOW, RELATIVE, false) -> branch
// when OVERFLOW not set.
func (c *cpu) branch(mask uint8, predicate bool) {
	if (c.status&mask > 0) == predicate {
		c.pc = c.getOperandAddr(RELATIVE)
	}
}

func (c *cpu) opAND(mode uint8) {
	c.acc = c.acc & c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) opASL(mode uint8) {
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
		c.writeMem(addr, nv)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&0x80 != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) opBEQ(mode uint8) {
	c.branch(STATUS_FLAG_ZERO, true)
}

func (c *cpu) opBIT(mode uint8) {
	o := c.memRead(c.getOperandAddr(mode))

	c.flagsOff(STATUS_FLAG_NEGATIVE | STATUS_FLAG_OVERFLOW | STATUS_FLAG_ZERO)
	var flags uint8
	if (o & c.acc) == 0 {
		flags = flags | STATUS_FLAG_ZERO
	}
	flags = flags | (o & (STATUS_FLAG_NEGATIVE | STATUS_FLAG_OVERFLOW))

	c.flagsOn(flags)
}

func (c *cpu) opBMI(mode uint8) {
	c.branch(STATUS_FLAG_NEGATIVE, true)
}

func (c *cpu) opBNE(mode uint8) {
	c.branch(STATUS_FLAG_ZERO, false)
}

func (c *cpu) opBPL(mode uint8) {
	c.branch(STATUS_FLAG_NEGATIVE, false)
}

func (c *cpu) opBVC(mode uint8) {
	c.branch(STATUS_FLAG_OVERFLOW, false)
}

func (c *cpu) opBVS(mode uint8) {
	c.branch(STATUS_FLAG_OVERFLOW, true)
}

func (c *cpu) opCLC(mode uint8) {
	c.flagsOff(STATUS_FLAG_CARRY)
}

func (c *cpu) opCLD(mode uint8) {
	c.flagsOff(STATUS_FLAG_DECIMAL)
}

func (c *cpu) opCLI(mode uint8) {
	c.flagsOff(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) opCLV(mode uint8) {
	c.flagsOff(STATUS_FLAG_OVERFLOW)
}

func (c *cpu) opDEC(mode uint8) {
	a := c.getOperandAddr(mode)
	c.writeMem(a, c.memRead(a)-1)
	c.setNegativeAndZeroFlags(c.memRead(a))
}

func (c *cpu) opDEX(mode uint8) {
	c.x -= 1
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) opDEY(mode uint8) {
	c.y -= 1
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) opEOR(mode uint8) {
	c.acc = c.acc ^ c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) opINC(mode uint8) {
	a := c.getOperandAddr(mode)
	c.writeMem(a, c.memRead(a)+1)
	c.setNegativeAndZeroFlags(c.memRead(a))
}

func (c *cpu) opINX(mode uint8) {
	c.x += 1
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) opINY(mode uint8) {
	c.y += 1
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) opJMP(mode uint8) {
	c.pc = c.memRead16(c.getOperandAddr(mode))
}

func (c *cpu) opJSR(mode uint8) {
	c.pushAddress(c.pc - 1)
	c.pc = c.getOperandAddr(mode)
}

func (c *cpu) opLDA(mode uint8) {
	c.acc = c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) opLDX(mode uint8) {
	c.x = c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) opLDY(mode uint8) {
	c.y = c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) opLSR(mode uint8) {
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
		c.writeMem(addr, nv)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&STATUS_FLAG_CARRY != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}

}

func (c *cpu) opNOP(mode uint8) {
	return
}

func (c *cpu) opORA(mode uint8) {
	c.acc = c.acc | c.memRead(c.getOperandAddr(mode))
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) opPHA(mode uint8) {
	c.pushStack(c.acc)
}

func (c *cpu) opPHP(mode uint8) {
	c.pushStack(c.status)
}

func (c *cpu) opPLA(mode uint8) {
	c.acc = c.popStack()
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) opPLP(mode uint8) {
	c.status = c.popStack()
}

func (c *cpu) opROL(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = bits.RotateLeft8(ov, 1) | (c.status & STATUS_FLAG_CARRY)
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.memRead(addr)
		c.writeMem(addr, bits.RotateLeft8(ov, 1)|(c.status&STATUS_FLAG_CARRY))
		nv = c.memRead(addr)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&0x80 != 0 {
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) opROR(mode uint8) {
	var ov, nv uint8 // old value, new value
	switch mode {
	case ACCUMULATOR:
		ov = c.acc
		c.acc = bits.RotateLeft8(ov, -1) | ((c.status & STATUS_FLAG_CARRY) << 7)
		nv = c.acc
	default:
		addr := c.getOperandAddr(mode)
		ov = c.memRead(addr)
		c.writeMem(addr, bits.RotateLeft8(ov, -1)|((c.status&STATUS_FLAG_CARRY)<<7))
		nv = c.memRead(addr)
	}

	c.flagsOff(STATUS_FLAG_CARRY | STATUS_FLAG_NEGATIVE | STATUS_FLAG_ZERO)
	c.setNegativeAndZeroFlags(nv)
	if ov&STATUS_FLAG_CARRY != 0 { // was carry bit set in the old _value_?
		c.flagsOn(STATUS_FLAG_CARRY)
	}
}

func (c *cpu) opRTS(mode uint8) {
	c.pc = c.popAddress() + 1
}

func (c *cpu) opSEC(mode uint8) {
	c.flagsOn(STATUS_FLAG_CARRY)
}

func (c *cpu) opSED(mode uint8) {
	c.flagsOn(STATUS_FLAG_DECIMAL)
}

func (c *cpu) opSEI(mode uint8) {
	c.flagsOn(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) opSTA(mode uint8) {
	c.writeMem(c.getOperandAddr(mode), c.acc)
}

func (c *cpu) opSTX(mode uint8) {
	c.writeMem(c.getOperandAddr(mode), c.x)
}

func (c *cpu) opSTY(mode uint8) {
	c.writeMem(c.getOperandAddr(mode), c.y)
}

func (c *cpu) opTAX(mode uint8) {
	c.x = c.acc
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) opTAY(mode uint8) {
	c.y = c.acc
	c.setNegativeAndZeroFlags(c.y)
}

func (c *cpu) opTSX(mode uint8) {
	c.x = c.sp
	c.setNegativeAndZeroFlags(c.x)
}

func (c *cpu) opTXA(mode uint8) {
	c.acc = c.x
	c.setNegativeAndZeroFlags(c.acc)
}

func (c *cpu) opTXS(mode uint8) {
	c.sp = c.x
}

func (c *cpu) opTYA(mode uint8) {
	c.acc = c.y
	c.setNegativeAndZeroFlags(c.acc)
}

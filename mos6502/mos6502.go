// Package mos6502 implements the MOS Technologies 6502 processor
// https://en.wikipedia.org/wiki/MOS_Technology_6502
package mos6502

import (
	"errors"
	"fmt"
	"math"
)

// 6502 Processor Status Flags
// https://www.nesdev.org/obelisk-6502-guide/registers.html
const (
	STATUS_FLAG_CARRY             = iota // C
	STATUS_FLAG_ZERO                     // Z
	STATUS_FLAG_INTERRUPT_DISABLE        // I
	STATUS_FLAG_DECIMAL                  // D
	STATUS_FLAG_BREAK                    // B
	STATUS_FLAG_UNUSED                   // -
	STATUS_FLAG_OVERFLOW                 // V
	STATUS_FLAG_NEGATIVE                 // N
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
	inst uint8 // The instruction
	mode uint8 // The memory addressing mode to use
}

func (o opcode) String() string {
	return fmt.Sprintf("{%s, %s}", opnames[o.inst], modenames[o.mode])
}

var opcodes map[uint8]opcode = map[uint8]opcode{
	// ADC
	0x69: opcode{ADC, IMMEDIATE},
	0x65: opcode{ADC, ZERO_PAGE},
	0x75: opcode{ADC, ZERO_PAGE_X},
	0x6D: opcode{ADC, ABSOLUTE},
	0x7D: opcode{ADC, ABSOLUTE_X},
	0x79: opcode{ADC, ABSOLUTE_Y},
	0x61: opcode{ADC, INDIRECT_X},
	0x71: opcode{ADC, INDIRECT_Y},
	// AND
	0x29: opcode{AND, IMMEDIATE},
	0x25: opcode{AND, ZERO_PAGE},
	0x35: opcode{AND, ZERO_PAGE_X},
	0x2D: opcode{AND, ABSOLUTE},
	0x3D: opcode{AND, ABSOLUTE_X},
	0x39: opcode{AND, ABSOLUTE_Y},
	0x21: opcode{AND, INDIRECT_X},
	0x31: opcode{AND, INDIRECT_Y},
	// ASL
	0x0A: opcode{ASL, ACCUMULATOR},
	0x06: opcode{ASL, ZERO_PAGE},
	0x16: opcode{ASL, ZERO_PAGE_X},
	0x0E: opcode{ASL, ABSOLUTE},
	0x1E: opcode{ASL, ABSOLUTE_X},
	// BCC
	0x90: opcode{BCC, RELATIVE},
	// BCS
	0xB0: opcode{BCS, RELATIVE},
	// BEQ
	0xF0: opcode{BEQ, RELATIVE},
	// BIT
	0x24: opcode{BIT, ZERO_PAGE},
	0x2C: opcode{BIT, ABSOLUTE},
	// BMI
	0x30: opcode{BMI, RELATIVE},
	// BNE
	0xD0: opcode{BNE, RELATIVE},
	// BPL
	0x10: opcode{BPL, RELATIVE},
	// BRK
	0x00: opcode{BRK, IMPLICIT},
	// BVC
	0x50: opcode{BVC, RELATIVE},
	// BVS
	0x70: opcode{BVS, RELATIVE},
	// CLC
	0x18: opcode{CLC, IMPLICIT},
	// CLD
	0xD8: opcode{CLD, IMPLICIT},
	// CLI
	0x58: opcode{CLI, IMPLICIT},
	// CLV
	0xB8: opcode{CLV, IMPLICIT},
	// CMP
	0xC9: opcode{CMP, IMMEDIATE},
	0xC5: opcode{CMP, ZERO_PAGE},
	0xD5: opcode{CMP, ZERO_PAGE_X},
	0xCD: opcode{CMP, ABSOLUTE},
	0xDD: opcode{CMP, ABSOLUTE_X},
	0xD9: opcode{CMP, ABSOLUTE_Y},
	0xC1: opcode{CMP, INDIRECT_X},
	0xD1: opcode{CMP, INDIRECT_Y},
	// CPX
	0xE0: opcode{CPX, IMMEDIATE},
	0xE4: opcode{CPX, ZERO_PAGE},
	0xEC: opcode{CPX, ABSOLUTE},
	// CPY
	0xC0: opcode{CPY, IMMEDIATE},
	0xC4: opcode{CPY, ZERO_PAGE},
	0xCC: opcode{CPY, ABSOLUTE},
	// DEC
	0xC6: opcode{DEC, ZERO_PAGE},
	0xD6: opcode{DEC, ZERO_PAGE_X},
	0xCE: opcode{DEC, ABSOLUTE},
	0xDE: opcode{DEC, ABSOLUTE_X},
	// DEX
	0xCA: opcode{DEX, IMPLICIT},
	// DEY
	0x88: opcode{DEY, IMPLICIT},
	// EOR
	0x49: opcode{EOR, IMMEDIATE},
	0x45: opcode{EOR, ZERO_PAGE},
	0x55: opcode{EOR, ZERO_PAGE_X},
	0x4D: opcode{EOR, ABSOLUTE},
	0x5D: opcode{EOR, ABSOLUTE_X},
	0x59: opcode{EOR, ABSOLUTE_Y},
	0x41: opcode{EOR, INDIRECT_X},
	0x51: opcode{EOR, INDIRECT_Y},
	// INC
	0xE6: opcode{INC, ZERO_PAGE},
	0xF6: opcode{INC, ZERO_PAGE_X},
	0xEE: opcode{INC, ABSOLUTE},
	0xFE: opcode{INC, ABSOLUTE_X},
	// INX
	0xE8: opcode{INX, IMPLICIT},
	// INY
	0xC8: opcode{INY, IMPLICIT},
	// JMP
	0x4C: opcode{JMP, ABSOLUTE},
	0x6C: opcode{JMP, INDIRECT},
	// JSR
	0x20: opcode{JSR, ABSOLUTE},
	// LDA
	0xA9: opcode{LDA, IMMEDIATE},
	0xA5: opcode{LDA, ZERO_PAGE},
	0xB5: opcode{LDA, ZERO_PAGE_X},
	0xAD: opcode{LDA, ABSOLUTE},
	0xBD: opcode{LDA, ABSOLUTE_X},
	0xB9: opcode{LDA, ABSOLUTE_Y},
	0xA1: opcode{LDA, INDIRECT_X},
	0xB1: opcode{LDA, INDIRECT_Y},
	// LDX
	0xA2: opcode{LDX, IMMEDIATE},
	0xA6: opcode{LDX, ZERO_PAGE},
	0xB6: opcode{LDX, ZERO_PAGE_Y},
	0xAE: opcode{LDX, ABSOLUTE},
	0xBE: opcode{LDX, ABSOLUTE_Y},
	// LDY
	0xA0: opcode{LDY, IMMEDIATE},
	0xA4: opcode{LDY, ZERO_PAGE},
	0xB4: opcode{LDY, ZERO_PAGE_X},
	0xAC: opcode{LDY, ABSOLUTE},
	0xBC: opcode{LDY, ABSOLUTE_X},
	// LSR
	0x4A: opcode{LSR, ACCUMULATOR},
	0x46: opcode{LSR, ZERO_PAGE},
	0x56: opcode{LSR, ZERO_PAGE_X},
	0x4E: opcode{LSR, ABSOLUTE},
	0x5E: opcode{LSR, ABSOLUTE_X},
	// NOP
	0xEA: opcode{NOP, IMPLICIT},
	// ORA
	0x09: opcode{ORA, IMMEDIATE},
	0x05: opcode{ORA, ZERO_PAGE},
	0x15: opcode{ORA, ZERO_PAGE_X},
	0x0D: opcode{ORA, ABSOLUTE},
	0x1D: opcode{ORA, ABSOLUTE_X},
	0x19: opcode{ORA, ABSOLUTE_Y},
	0x01: opcode{ORA, INDIRECT_X},
	0x11: opcode{ORA, INDIRECT_Y},
	// PHA
	0x48: opcode{PHA, IMPLICIT},
	// PHP
	0x08: opcode{PHP, IMPLICIT},
	// PLA
	0x68: opcode{PLA, IMPLICIT},
	// PLP
	0x28: opcode{PLP, IMPLICIT},
	// ROL
	0x2A: opcode{ROL, ACCUMULATOR},
	0x26: opcode{ROL, ZERO_PAGE},
	0x36: opcode{ROL, ZERO_PAGE_X},
	0x2E: opcode{ROL, ABSOLUTE},
	0x3E: opcode{ROL, ABSOLUTE_X},
	// ROR
	0x6A: opcode{ROR, ACCUMULATOR},
	0x66: opcode{ROR, ZERO_PAGE},
	0x76: opcode{ROR, ZERO_PAGE_X},
	0x6E: opcode{ROR, ABSOLUTE},
	0x7E: opcode{ROR, ABSOLUTE_X},
	// RTI
	0x40: opcode{RTI, IMPLICIT},
	// RTS
	0x60: opcode{RTS, IMPLICIT},
	// SBC
	0xE9: opcode{SBC, IMMEDIATE},
	0xE5: opcode{SBC, ZERO_PAGE},
	0xF5: opcode{SBC, ZERO_PAGE_X},
	0xED: opcode{SBC, ABSOLUTE},
	0xFD: opcode{SBC, ABSOLUTE_X},
	0xF9: opcode{SBC, ABSOLUTE_Y},
	0xE1: opcode{SBC, INDIRECT_X},
	0xF1: opcode{SBC, INDIRECT_Y},
	// SEC
	0x38: opcode{SEC, IMPLICIT},
	// SED
	0xF8: opcode{SED, IMPLICIT},
	// SEI
	0x78: opcode{SEI, IMPLICIT},
	// STA
	0x85: opcode{STA, ZERO_PAGE},
	0x95: opcode{STA, ZERO_PAGE_X},
	0x8D: opcode{STA, ABSOLUTE},
	0x9D: opcode{STA, ABSOLUTE_X},
	0x99: opcode{STA, ABSOLUTE_Y},
	0x81: opcode{STA, INDIRECT_X},
	0x91: opcode{STA, INDIRECT_Y},
	// STX
	0x86: opcode{STX, ZERO_PAGE},
	0x96: opcode{STX, ZERO_PAGE_Y},
	0x8E: opcode{STX, ABSOLUTE},
	// STY
	0x84: opcode{STY, ZERO_PAGE},
	0x94: opcode{STY, ZERO_PAGE_X},
	0x8C: opcode{STY, ABSOLUTE},
	// TAX
	0xAA: opcode{TAX, IMPLICIT},
	// TAY
	0xA8: opcode{TAY, IMPLICIT},
	// TSX
	0xBA: opcode{TSX, IMPLICIT},
	// TXA
	0x8A: opcode{TXA, IMPLICIT},
	// TXS:
	0x9A: opcode{TXS, IMPLICIT},
	// TYA
	0x98: opcode{TYA, IMPLICIT},
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
	m := c.memory[c.pc]
	op, ok := opcodes[m]
	if !ok {
		return opcodes[0x00], fmt.Errorf("pc: %d, inst: 0x%02x - %w", c.pc, m, invalidInstruction)
	}

	return op, nil
}

func (c *cpu) step() {
	op, err := c.getInst()
	if err != nil {
		panic(err)
	}

	switch op.inst {
	case SEC:
		c.opSEC(op.mode)
	case SED:
		c.opSED(op.mode)
	case SEI:
		c.opSEI(op.mode)
	case CLC:
		c.opCLC(op.mode)
	case CLD:
		c.opCLD(op.mode)
	case CLI:
		c.opCLI(op.mode)
	case CLV:
		c.opCLV(op.mode)
	default:
		panic(fmt.Errorf("unimplemented instruction %s", op))
	}
}

// flagOn forces the STATUS_FLAG_??? passed as flag on in the status
// register.
func (c *cpu) flagOn(flag uint8) {
	c.status = c.status | uint8(1<<flag)
}

// flagOff forces the STATUS_FLAG_??? passed as flag off in the status
// register.
func (c *cpu) flagOff(flag uint8) {
	c.status = c.status &^ uint8(1<<flag)
}

func (c *cpu) opSEC(mode uint8) {
	c.flagOn(STATUS_FLAG_CARRY)
}

func (c *cpu) opSED(mode uint8) {
	c.flagOn(STATUS_FLAG_DECIMAL)
}

func (c *cpu) opSEI(mode uint8) {
	c.flagOn(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) opCLC(mode uint8) {
	c.flagOff(STATUS_FLAG_CARRY)
}

func (c *cpu) opCLD(mode uint8) {
	c.flagOff(STATUS_FLAG_DECIMAL)
}

func (c *cpu) opCLI(mode uint8) {
	c.flagOff(STATUS_FLAG_INTERRUPT_DISABLE)
}

func (c *cpu) opCLV(mode uint8) {
	c.flagOff(STATUS_FLAG_OVERFLOW)
}

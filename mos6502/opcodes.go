package mos6502

import (
	"fmt"
)

// 6502 Addressing Modes
// https://www.nesdev.org/obelisk-6502-guide/addressing.html
const (
	IMPLICIT = iota
	ACCUMULATOR
	IMMEDIATE
	ZERO_PAGE
	ZERO_PAGE_X
	ZERO_PAGE_X_BUT_Y // undocumented mode; https://www.nesdev.org/6502_cpu.txt
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
	LAX        // Load ACC and X from memory, undocumented
	SAX        // And X = (ACC & X) - immediate value, undocumented
	DCM        // m--; cmp acc w/m - undocumented
	ISB        // m++; acc - m - undocumented
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
	0x04: opcode{NOP, "NOP", ZERO_PAGE, 2, 2},   // undocumented
	0x44: opcode{NOP, "NOP", ZERO_PAGE, 2, 2},   // undocumented
	0x64: opcode{NOP, "NOP", ZERO_PAGE, 2, 2},   // undocumented
	0x0c: opcode{NOP, "NOP", ABSOLUTE, 2, 2},    // undocumented
	0x14: opcode{NOP, "NOP", ZERO_PAGE_X, 2, 2}, // undocumented
	0x34: opcode{NOP, "NOP", ZERO_PAGE_X, 2, 2}, // undocumented
	0x54: opcode{NOP, "NOP", ZERO_PAGE_X, 2, 2}, // undocumented
	0x74: opcode{NOP, "NOP", ZERO_PAGE_X, 2, 2}, // undocumented
	0xD4: opcode{NOP, "NOP", ZERO_PAGE_X, 2, 2}, // undocumented
	0xF4: opcode{NOP, "NOP", ZERO_PAGE_X, 2, 2}, // undocumented
	0xEA: opcode{NOP, "NOP", IMPLICIT, 1, 2},
	0x1A: opcode{NOP, "NOP", IMPLICIT, 2, 2},   // undocumented
	0x3A: opcode{NOP, "NOP", IMPLICIT, 2, 2},   // undocumented
	0x5A: opcode{NOP, "NOP", IMPLICIT, 2, 2},   // undocumented
	0xDA: opcode{NOP, "NOP", IMPLICIT, 2, 2},   // undocumented
	0x80: opcode{NOP, "NOP", IMPLICIT, 2, 2},   // undocumented
	0x1C: opcode{NOP, "NOP", ABSOLUTE_X, 2, 2}, // undocumented
	0x3C: opcode{NOP, "NOP", ABSOLUTE_X, 2, 2}, // undocumented
	0x5C: opcode{NOP, "NOP", ABSOLUTE_X, 2, 2}, // undocumented
	0x7C: opcode{NOP, "NOP", ABSOLUTE_X, 2, 2}, // undocumented
	0xDC: opcode{NOP, "NOP", ABSOLUTE_X, 2, 2}, // undocumented
	0xFC: opcode{NOP, "NOP", ABSOLUTE_X, 2, 2}, // undocumented
	0x09: opcode{ORA, "ORA", IMMEDIATE, 2, 2},
	0x05: opcode{ORA, "ORA", ZERO_PAGE, 2, 3},
	0x15: opcode{ORA, "ORA", ZERO_PAGE_X, 2, 4},
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
	0xEB: opcode{SBC, "SBC", IMMEDIATE, 2, 2}, // undocumented
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
	0xA3: opcode{LAX, "LAX", INDIRECT_X, 2, 6},
	0xB3: opcode{LAX, "LAX", INDIRECT_Y, 2, 5},
	0xBF: opcode{LAX, "LAX", ABSOLUTE_Y, 3, 4},
	0xAF: opcode{LAX, "LAX", ABSOLUTE, 3, 4},
	0xB7: opcode{LAX, "LAX", ZERO_PAGE_Y, 2, 4},
	0xA7: opcode{LAX, "LAX", ZERO_PAGE_Y, 2, 3},
	0x83: opcode{SAX, "SAX", IMMEDIATE, 2, 2},
	0x87: opcode{SAX, "SAX", ZERO_PAGE, 2, 3},
	0x8f: opcode{SAX, "SAX", ABSOLUTE, 2, 4},
	0x97: opcode{SAX, "SAX", ZERO_PAGE_X_BUT_Y, 2, 4},
	0xCF: opcode{DCM, "DCM", ABSOLUTE, 3, 6},
	0xDF: opcode{DCM, "DCM", ABSOLUTE_X, 3, 7},
	0xDB: opcode{DCM, "DCM", ABSOLUTE_Y, 3, 7},
	0xC7: opcode{DCM, "DCM", ZERO_PAGE, 2, 5},
	0xD7: opcode{DCM, "DCM", ZERO_PAGE_X, 2, 6},
	0xC3: opcode{DCM, "DCM", INDIRECT_X, 2, 8},
	0xD3: opcode{DCM, "DCM", INDIRECT_Y, 2, 8},
	0xEF: opcode{ISB, "ISB", ABSOLUTE, 3, 6},
	0xFF: opcode{ISB, "ISB", ABSOLUTE_X, 3, 7},
	0xFB: opcode{ISB, "ISB", ABSOLUTE_Y, 3, 7},
	0xE7: opcode{ISB, "ISB", ZERO_PAGE, 2, 5},
	0xF7: opcode{ISB, "ISB", ZERO_PAGE_X, 2, 6},
	0xE3: opcode{ISB, "ISB", INDIRECT_X, 2, 8},
	0xF3: opcode{ISB, "ISB", INDIRECT_Y, 2, 8},
}

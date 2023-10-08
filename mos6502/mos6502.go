// Package mos6502 implements the MOS Technologies 6502 processor
// https://en.wikipedia.org/wiki/MOS_Technology_6502
package mos6502

import (
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
}

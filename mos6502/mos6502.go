// package mos6502 implements the MOS Technologies 6502 processor
package mos6502

import (
	"math"
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

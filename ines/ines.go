// package ines implements support for the NES (iNES) ROM format
// https://www.nesdev.org/wiki/INES
package ines

import (
	"fmt"
	"io"
)

type Header struct {
	// Bytes 0-3
	// Constant $4E $45 $53 $1A (ASCII "NES" followed by MS-DOS end-of-file)
	constant string
	// Byte 4
	// Size of PRG ROM in 16 KB units
	prgSize uint8
	// Byte 5
	// Size of CHR ROM in 8 KB units (value 0 means the board uses CHR RAM)
	chrSize uint8
	// Byte 6
	// Flags 6 – Mapper, mirroring, battery, trainer
	flags6 uint8
	// Byte 7
	// Flags 7 – Mapper, VS/Playchoice, NES 2.0
	flags7 uint8
	// Byte 8
	// Flags 8 – PRG-RAM size (rarely used extension)
	flags8 uint8
	// Byte 9
	// Flags 9 – TV system (rarely used extension)
	flags9 uint8
	// Byte 10
	// Flags 10 – TV system, PRG-RAM presence (unofficial, rarely used extension)
	flags10 uint8
	// Bytes 11-15	Unused padding (should be filled with zero, but some rippers put their name across bytes 7-15)
	unused string
}



type PlayChoicePROM struct {
	Data       [16]byte
	CounterOut [16]byte
}

type nesRom struct {
	header    *Header
	trainer   []byte          // if present
	prg       []byte          // 16384 * x bytes; x from header
	chr       []byte          // 8192 * y bytes; y from header
	pcInstRom []byte          // if present
	pcPROM    *PlayChoicePROM // if present; often missing - see PC10 ROM-Images
}
}

func parseHeader(hbytes []byte) *Header {
	return &Header{
		constant: string(hbytes[0:4]),
		prgSize:  uint8(hbytes[4]),
		chrSize:  uint8(hbytes[5]),
		flags6:   uint8(hbytes[6]),
		flags7:   uint8(hbytes[7]),
		flags8:   uint8(hbytes[8]),
		flags9:   uint8(hbytes[9]),
		flags10:  uint8(hbytes[10]),
	}
}

func New(inesData io.Reader) (*nesRom, error) {
	hbytes := make([]byte, 16)
	n, err := inesData.Read(hbytes)
	if n != 16 || err != nil {
		return nil, fmt.Errorf("couldn't read header: %w", err)
	}

	return &nesROM{header: parseHeader(hbytes)}, nil
}

func (h *Header) String() string {
	return fmt.Sprintf("%s, prg(%d), chr(%d), flags(%d, %d, %d, %d, %d)", h.constant, h.prgSize, h.chrSize, h.flags6, h.flags7, h.flags8, h.flags9, h.flags10)
}

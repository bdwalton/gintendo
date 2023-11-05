// package nesFormat implements support for the NES (iNES) ROM format
// https://www.nesdev.org/wiki/INES, https://www.nesdev.org/wiki/NES_2.0
package nesrom

import (
	"fmt"
)

type header struct {
	// Bytes 0-3
	// Constant $4E $45 $53 $1A (ASCII "NES" followed by MS-DOS end-of-file)
	constant string
	// Byte 4
	// iNES: Size of PRG ROM in 16 KB units
	// NES2: PRG-ROM size, LSB
	prgSize uint8
	// Byte 5
	// iNES: Size of CHR ROM in 8 KB units (value 0 means the board uses CHR RAM)
	// NES2: CHR-ROM size, LSB
	chrSize uint8
	// Byte 6
	// Flags 6 – Mapper, mirroring, battery, trainer
	flags6 uint8
	// Byte 7
	// Flags 7 – Mapper, VS/Playchoice, NES 2.0
	flags7 uint8
	// Byte 8
	// iNES: Flags 8 – PRG-RAM size (rarely used extension)
	// NES2: Mapper MSB/Submapper
	flags8 uint8
	// Byte 9
	// iNES: Flags 9 – TV system (rarely used extension)
	// NES2: PRG-ROM/CHR-ROM size, MSB
	flags9 uint8
	// Byte 10
	// iNES: Flags 10 – TV system, PRG-RAM presence (unofficial, rarely used extension)
	// NES2: PRG-RAM/EEPROM size
	flags10 uint8
	// iNES: Unused
	// NES2: CHR-RAM size
	flags11 uint8
	// iNES: Unused
	// NES2: CPU/PPU Timing
	flags12 uint8
	// iNES: Unused
	// NES2: When Byte 7 AND 3 =1: Vs. System Type
	flags13 uint8
	// iNES: Unused
	// NES2: Miscellaneous ROMs
	flags14 uint8
	// iNES: Unused
	// NES2: Default Expansion Device
	flags15 uint8
}

// flag6 flag identifiers - the top 4 bits are the lower nibble of the mapper number
const (
	// 0: horizontal (vertical arrangement) (CIRAM A10 = PPU A11)
	// 1: vertical (horizontal arrangement) (CIRAM A10 = PPU A10)
	MIRRORING = 1 << 0
	// 1: Cartridge contains battery-backed PRG RAM ($6000-7FFF)
	// or other persistent memory
	BATTERY_BACKED_SRAM = 1 << 1
	// 1: 512-byte trainer at $7000-$71FF (stored before PRG data)
	TRAINER = 1 << 2
	// 1: Ignore mirroring control or above mirroring bit; instead
	// provide four-screen VRAM
	IGNORE_MIRRORING = 1 << 3
)

// flag7 flag identifiers - the top 4 bits are the upper nibble of the mapper number
const (
	VS_UNISYSTEM = 0x01
	// Mostly ignored
	PLAYCHOICE_10 = 0x02 // PlayChoice-10, 8 KB of Hint Screen data stored after CHR data
)

// flags9 flag identifiers
const (
	TV_SYSTEM = 0x01
)

func (h *header) String() string {
	return fmt.Sprintf("%s, prg(%d), chr(%d), flags(%02x, %02x, %02x, %02x, %02x)", h.constant, h.prgSize, h.chrSize, h.flags6, h.flags7, h.flags8, h.flags9, h.flags10)
}

// Mirroring mode
const (
	MIRROR_HORIZONTAL = iota
	MIRROR_VERTICAL
	MIRROR_FOUR_SCREEN
)

// mirroringMode returns an identifier indicating which mirroring mode
// the PPU should use during rendering.
// https://www.nesdev.org/wiki/INES#Nametable_Mirroring
func (h *header) mirroringMode() uint8 {
	if h.flags6&IGNORE_MIRRORING > 0 {
		return MIRROR_FOUR_SCREEN
	}

	return h.flags6 & MIRRORING // 0 = horizonal, 1 = vertical
}

// hasTrainer indicates whether the NES ROM contains a Trainer
func (h *header) hasTrainer() bool {
	return h.flags6&TRAINER == TRAINER
}

func (h *header) hasPlayChoice() bool {
	return h.flags7&PLAYCHOICE_10 == PLAYCHOICE_10
}

func (h *header) hasPrgRAM() bool {
	return h.flags6&BATTERY_BACKED_SRAM > 0
}

// PrgRAMSize returns the size of PRG RAM in 8KB units with flags8==0
// indicating that there is a single (1) 8KB unit
func (h *header) prgRAMSize() uint8 {
	if h.hasPrgRAM() {
		if h.flags8 == 0 {
			return 1
		}

		return h.flags8
	}

	return 0
}

const (
	NTSC = iota
	PAL
)

func (h *header) tvSystem() uint8 {
	return h.flags9 & TV_SYSTEM
}

func (h *header) isINesFormat() bool {
	return h.constant == "NES\x1A"
}

func (h *header) isNES2Format() bool {
	return h.isINesFormat() && ((h.flags7 & 0x0C) == 0x08)
}

// ignoreHighNibble returns true if we should not use the high 4 bits of
// Older versions of the iNES emulator ignored bytes 7-15, and several
// ROM management tools wrote messages in there. Commonly, these will
// be filled with "DiskDude!", which results in 64 being added to the
// mapper number. A general rule of thumb: if the last 4 bytes are not
// all zero, and the header is not marked for NES 2.0 format, an
// emulator should either mask off the upper 4 bits of the mapper
// number or simply refuse to load the ROM.
func (h *header) ignoreHighNibble() bool {
	lfbz := true // last 4 bytes zero
	for _, x := range []byte{h.flags12, h.flags13, h.flags14, h.flags15} {
		if x != 0x00 {
			lfbz = false
			break
		}
	}

	if !lfbz && !h.isNES2Format() {
		return true
	}

	return false
}

// mapperNum returns the mapper number which is constructed of the
// upper 4 bits of flag7 and the upper 4 bits of flag 6.
func (h *header) mapperNum() uint16 {
	// iNES and NES2 share the first 8 bits of the mapper number in the flags
	mn := ((h.flags6 & 0xF0) >> 4)
	if !h.ignoreHighNibble() {
		mn = (h.flags7 & 0xF0) | mn
	}

	if h.isNES2Format() {
		// Add the additional 4 bits of mapper number identifier
		return (uint16(h.flags8&0xF) << 8) | uint16(mn)
	}

	return uint16(mn)
}

func parseHeader(hbytes []byte) *header {
	return &header{
		constant: string(hbytes[0:4]),
		prgSize:  uint8(hbytes[4]),
		chrSize:  uint8(hbytes[5]),
		flags6:   uint8(hbytes[6]),
		flags7:   uint8(hbytes[7]),
		flags8:   uint8(hbytes[8]),
		flags9:   uint8(hbytes[9]),
		flags10:  uint8(hbytes[10]),
		flags11:  uint8(hbytes[11]),
		flags12:  uint8(hbytes[12]),
		flags13:  uint8(hbytes[13]),
		flags14:  uint8(hbytes[14]),
	}
}

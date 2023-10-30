package console

import (
	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/nesrom"
)

type ppuMemory struct {
	size   uint16  // The size of vram in words
	ram    []uint8 // The actual vram memory
	mapper mappers.Mapper
}

func newPPUMemory(size uint16, m mappers.Mapper) *ppuMemory {
	return &ppuMemory{size: size, ram: make([]uint8, size), mapper: m}
}

const (
	PATTERN_TABLE_0  = 0x0000
	PATTERN_TABLE_1  = 0x1000
	NAMETABLE_0      = 0x2000
	NAMETABLE_1      = 0x2400
	NAMETABLE_2      = 0x2800
	NAMETABLE_3      = 0x2C00
	NAMETABLE_MIRROR = 0x3000
	PALETTE_RAM      = 0x3F00
	PALETTE_MIRROR   = 0x3F20
)

func tileMapAddr(addr uint16, mm uint8) uint16 {
	// Now we have a as the base of our internal memory
	a := addr - NAMETABLE_0
	// https://www.nesdev.org/wiki/Mirroring#Nametable_Mirroring
	switch mm {
	case nesrom.MIRROR_FOUR_SCREEN:
		panic("we don't have mapper support to leverage vram on catridge")
	case nesrom.MIRROR_HORIZONTAL:
		if a >= 0x800 {
			return 0x0400 + ((a - 0x800) % 0x400)
		}
		return a % 0x0400
	case nesrom.MIRROR_VERTICAL:
		return a % 0x800
	}

	panic("unkown mirroring mode")
}

// topRangeMap returns the mirrored address when the access is above 0x4000
func topRangeMap(addr uint16) uint16 {
	if addr >= 0x4000 {
		return addr % 0x4000
	}

	return addr
}

func (m *ppuMemory) read(addr uint16) uint8 {
	a := topRangeMap(addr)

	switch {
	case a < NAMETABLE_0:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		return m.mapper.ChrRead(a)
	case a < PALETTE_RAM:
		return m.ram[tileMapAddr(a, m.mapper.MirroringMode())]
	case a < PALETTE_MIRROR:
		return m.ram[a-PALETTE_RAM]
	default:
		x := (a - PALETTE_RAM) % 0x0020
		return m.ram[PALETTE_RAM+x]
	}
}

func (m *ppuMemory) write(addr uint16, val uint8) {
	a := topRangeMap(addr)

	switch {
	case a < NAMETABLE_0:
		// Pattern Table 0 and 1 (upper: 0x0FFF, 0x1FFF)
		m.mapper.ChrWrite(a, val)
	case a < PALETTE_RAM:
		m.ram[tileMapAddr(a, m.mapper.MirroringMode())] = val
	case a < PALETTE_MIRROR:
		m.ram[a-PALETTE_RAM] = val
	default:
		x := (a - PALETTE_RAM) % 0x0020
		m.ram[PALETTE_RAM+x] = val
	}
}

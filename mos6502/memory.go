package mos6502

import (
	"math"

	"github.com/bdwalton/gintendo/mappers"
)

const (
	MAX_ADDRESS = math.MaxUint16
)

type memory struct {
	ram    []uint8        // The actual memory
	mapper mappers.Mapper // Access to "virtualized" memory via the mapper
}

func newMemory(ramSize uint16, m mappers.Mapper) *memory {
	return &memory{ram: make([]uint8, ramSize), mapper: m}
}

func (m *memory) read(addr uint16) uint8 {
	switch {
	case addr < uint16(len(m.ram)):
		return m.ram[addr]
	case addr <= MAX_ADDRESS:
		return m.mapper.PrgRead(addr)
	}

	panic("should never happen") // hah, prod crashes await!
}

func (m *memory) write(addr uint16, val uint8) {
	switch {
	case addr < uint16(len(m.ram)):
		m.ram[addr] = val
	case addr <= MAX_ADDRESS:
		m.mapper.PrgWrite(addr, val)
	}
}

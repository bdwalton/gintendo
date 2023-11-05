package console

import (
	"math"

	"github.com/bdwalton/gintendo/mappers"
)

const (
	MAX_ADDRESS         = math.MaxUint16
	MAX_IO_REG_MIRRORED = 0x4000
	MAX_IO_REG          = 0x4020
)

type cpuMemory struct {
	mach   *machine       // The machine, which encapsulates the wider memory bus
	size   uint16         // The size of ram in words
	ram    []uint8        // The actual memory
	mapper mappers.Mapper // Access to "virtualized" memory via the mapper
}

func newCPUMemory(mach *machine, size uint16, m mappers.Mapper) *cpuMemory {
	return &cpuMemory{
		mach:   mach,
		size:   size,
		ram:    make([]uint8, size),
		mapper: m}
}

func (m *cpuMemory) read(addr uint16) uint8 {
	// https://www.nesdev.org/wiki/CPU_memory_map
	switch {
	case addr <= 0x1FFF:
		// 0x800-0x1FFF mirrors 0x0000-0x07FF
		return m.ram[addr%m.size]
	case addr < MAX_IO_REG_MIRRORED:
		// PPU registers are mirrored between 0x2000 and 0x4000
		return m.mach.ReadPPU(0x2000 + ((addr - 0x2000) % 0x8))
	case addr < MAX_IO_REG:
		// handle joysticks
	case addr <= MAX_ADDRESS:
		return m.mapper.PrgRead(addr)
	}

	panic("should never happen") // hah, prod crashes await!
}

// read16 returns the two bytes from memory at addr (lower byte is
// first).
func (m *cpuMemory) read16(addr uint16) uint16 {
	lsb := uint16(m.read(addr))
	msb := uint16(m.read(addr + 1))

	return (msb << 8) | lsb
}

func (m *cpuMemory) write(addr uint16, val uint8) {
	// https://www.nesdev.org/wiki/CPU_memory_map
	switch {
	case addr <= 0x1FFF:
		// 0x800-0x1FFF mirrors 0x0000-0x07FF
		m.ram[addr%m.size] = val
	case addr < MAX_IO_REG_MIRRORED:
		// PPU registers are mirrored between 0x2000 and 0x4000
		m.mach.WritePPU(0x2000+((addr-0x2000)%0x8), val)
	case addr < MAX_IO_REG:
		// handle joysticks
	case addr <= MAX_ADDRESS:
		m.mapper.PrgWrite(addr, val)
	}
}

// write16 stores val at addr (lower byte is first).
func (m *cpuMemory) write16(addr, val uint16) {
	m.write(addr, uint8(val&0x00FF))
	m.write(addr+1, uint8(val>>8))
}

package mappers

func init() {
	m := newMapper0()
	RegisterMapper(m.ID(), m)
}

type mapper0 struct {
	*baseMapper
	prgRAM []uint8
}

func newMapper0() *mapper0 {
	return &mapper0{
		baseMapper: newBaseMapper(0, "NROM"),
		prgRAM:     make([]uint8, 0x7FFF-0x6000),
	}
}

func (m *mapper0) MemWrite(addr uint16, val uint8) {
	switch {
	case addr < NES_BASE_MEMORY:
		m.baseRAM[addr] = val
	case 0x0800 <= addr && addr <= 0x1FFF:
		// TODO: Maybe just make all of these mod
		// NES_BASE_MEMORY so there is 1 case instead of 2
		// here?  3 banks of $0800 addresses mapped back into
		// 3 banks of $0800 addresses mapped back into base memory
		m.baseRAM[(addr-0x0800)%NES_BASE_MEMORY] = val
	case 0x6000 <= addr && addr <= 0x7FFF:
		// PRG RAM
		m.prgRAM[addr-0x6000] = val
	case 0x8000 <= addr && addr <= 0xFFFF:
		// If we have two blocks of PRG, we can read higher
		// within the block, up to 32k. Otherwise, we map the
		// second 16k address range into the first so there is
		// mirroring.
		switch m.rom.NumPrgBlocks() {
		case 2:
			m.rom.PrgWrite(addr-0x8000, val)
		default:
			m.rom.PrgWrite(addr-0xC000, val)
		}
	}
}

func (m mapper0) MemRead(addr uint16) uint8 {
	switch {
	case addr < NES_BASE_MEMORY:
		return m.baseRAM[addr]
	case 0x0800 <= addr && addr <= 0x1FFF:
		// TODO: Maybe just make all of these mod
		// NES_BASE_MEMORY so there is 1 case instead of 2
		// here?  3 banks of $0800 addresses mapped back into
		// base memory
		return m.baseRAM[(addr-0x0800)%NES_BASE_MEMORY]
	case 0x6000 <= addr && addr <= 0x7FFF:
		// PRG RAM
		return m.prgRAM[addr-0x6000]
	case 0x8000 <= addr && addr <= 0xFFFF:
		// If we have two blocks of PRG, we can read higher
		// within the block, up to 32k. Otherwise, we map the
		// second 16k address range into the first so there is
		// mirroring.
		switch m.rom.NumPrgBlocks() {
		case 2:
			return m.rom.PrgRead(addr - 0x8000)
		default:
			return m.rom.PrgRead(addr - 0xC000)
		}
	}

	// We should never rely on this, but we need to return something
	return 0
}

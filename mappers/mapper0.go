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

func (m *mapper0) PrgWrite(addr uint16, val uint8) {
	panic("mapper0: Writing PRG Data.")
}

func (m *mapper0) PrgRead(addr uint16) uint8 {
	// If we have two blocks of PRG, we can read higher
	// within the block, up to 32k. Otherwise, we map the
	// second 16k address range into the first so there is
	// mirroring.
	a := addr - 0x8000
	switch m.rom.NumPrgBlocks() {
	case 1:
		m.rom.PrgRead(a % 0x4000)
	case 2:
		return m.rom.PrgRead(a)
	default:
		panic("mapper0: Reading above 32k of PRG Data.")
	}

	// Never reached
	panic("mapper0: PrgRead() doing bad things.")
}

func (m *mapper0) ChrRead(start, end uint16) []uint8 {
	return m.rom.ChrRead(start, end)
}

func (m *mapper0) ChrWrite(addr uint16, val uint8) {
	panic("mapper0: These ROMs don't support ChrWrite().")
}

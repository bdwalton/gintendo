package mappers

func init() {
	m := newMapper0()
	RegisterMapper(m.ID(), m)
}

type mapper0 struct {
	*baseMapper
}

func newMapper0() *mapper0 {
	return &mapper0{baseMapper: newBaseMapper(0, "NROM")}
}

func (m *mapper0) MemWrite(addr uint16, val uint8) {

}

func (m mapper0) MemRead(addr uint16) uint8 {
	switch {
	case addr < NES_BASE_MEMORY:
		return m.baseRAM[addr]
	case 0x8000 <= addr && addr <= 0xBFFF:
		m.rom.PrgRead(addr - 0x8000)
	case 0xC000 <= addr && addr <= 0xFFFF:
		m.rom.PrgRead(addr - 0xC000)
	}
	return 0
}

package console

import (
	"github.com/bdwalton/gintendo/mappers"
)

const (
	VRAM_SIZE    = 2048
	OAM_SIZE     = 256
	PALETTE_SIZE = 32
)

type PPU struct {
	bus          *bus
	m            mappers.Mapper
	paletteTable [PALETTE_SIZE]uint8
	oamData      [OAM_SIZE]uint8
	vram         [VRAM_SIZE]uint8
	ppuAddr      *addrReg
}

func newPPU(b *bus, m mappers.Mapper) *PPU {
	return &PPU{bus: b, m: m, ppuAddr: &addrReg{}}
}

func (p *PPU) writePPUADDR(val uint8) {
	p.ppuAddr.set(val)
}

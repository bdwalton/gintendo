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
	mach         *machine
	m            mappers.Mapper
	paletteTable [PALETTE_SIZE]uint8
	oamData      [OAM_SIZE]uint8
	vram         [VRAM_SIZE]uint8
	ppuAddr      *addrReg
}

func newPPU(mach *machine, m mappers.Mapper) *PPU {
	return &PPU{mach: mach, m: m, ppuAddr: &addrReg{}}
}

func (p *PPU) writePPUADDR(val uint8) {
	p.ppuAddr.set(val)
}

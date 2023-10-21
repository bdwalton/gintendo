// Package mappers implements and registers mappers that are
// referenced numerically by iNES and NES2.0 ROM files.
package mappers

import (
	"github.com/bdwalton/gintendo/nesrom"
)

// A global registry of mappers, keyed by mapper id
var AllMappers map[uint8]Mapper = map[uint8]Mapper{}

const (
	NES_BASE_MEMORY = 2048 // 2KB built in RAM
)

type Mapper interface {
	Init(*nesrom.ROM)
	Name() string
	MemWrite(uint16, uint8) // Write to uint8 to address uint16
	MemRead(uint16) uint8   // Read uint8 from address uint16
}

type baseMapper struct {
	rom  *nesrom.ROM
	name string
	//The base amount of NES RAM (2k) will be accessed here.
	baseRAM []uint8
}

func (bm *baseMapper) String() string {
	return bm.name
}

func (bm *baseMapper) Init(r *nesrom.ROM) {
	bm.rom = r
}

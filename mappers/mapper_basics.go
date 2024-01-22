// Package mappers implements and registers mappers that are
// referenced numerically by iNES and NES2.0 ROM files.
package mappers

import (
	"fmt"

	"github.com/bdwalton/gintendo/nesrom"
)

// A global registry of mappers, keyed by mapper id
var allMappers map[uint16]Mapper = map[uint16]Mapper{}

func RegisterMapper(id uint16, m Mapper) {
	if om, ok := allMappers[id]; ok {
		panic(fmt.Sprintf("Can't re-register mapper id %d. It's used by %q.", id, om.Name()))
	}
	allMappers[id] = m
}

// Load will instantiate an nesrom.Rom from romFile and return a
// mapper with the specified id or an error if we can't load the ROM
// or don't have a mapper for that id yet.
func Load(romFile string) (Mapper, error) {
	rom, err := nesrom.New(romFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't load ROM: %v", err)
	}

	id := rom.MapperNum()
	m, ok := allMappers[id]
	if !ok {
		return nil, fmt.Errorf("uknown mapper id %d", id)
	}

	m.Init(rom)
	return m, nil
}

type Mapper interface {
	ID() uint16
	Init(*nesrom.ROM)
	Name() string
	PrgRead(uint16) uint8   // Read PRG data
	PrgWrite(uint16, uint8) // Write PRG data
	ChrRead(uint16) uint8   // Read CHR data
	ChrWrite(uint16, uint8) // Write CHR data
	MirroringMode() uint8   // Which mirroring mode is tilemap data stored in
	HasSaveRAM() bool       // Whether or not the cartridge exposes Save RAM at 0x6000-0x7999
}

type baseMapper struct {
	id   uint16
	rom  *nesrom.ROM
	name string
}

func newBaseMapper(id uint16, name string) *baseMapper {
	return &baseMapper{
		id:   id,
		name: name,
	}
}

func (bm *baseMapper) ID() uint16 {
	return bm.id
}

func (bm *baseMapper) String() string {
	return bm.name
}

func (bm *baseMapper) Name() string {
	return bm.name
}

func (bm *baseMapper) Init(r *nesrom.ROM) {
	bm.rom = r
}

func (bm *baseMapper) MirroringMode() uint8 {
	return bm.rom.MirroringMode()
}

func (bm *baseMapper) HasSaveRAM() bool {
	return bm.rom.HasSaveRAM()
}

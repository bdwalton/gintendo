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

// Get returns a mapper with the specified id or an error if we don't
// have a mapper for that id yet.
func Get(rom *nesrom.ROM) (Mapper, error) {
	id := rom.MapperNum()
	m, ok := allMappers[id]
	if !ok {
		return nil, fmt.Errorf("uknown mapper id %d", id)
	}

	m.Init(rom)
	return m, nil
}

const (
	NES_BASE_MEMORY = 2048 // 2KB built in RAM
)

type Mapper interface {
	ID() uint16
	Init(*nesrom.ROM)
	Name() string
	ReadBaseRAM(uint16) uint8   // Read from 2k Base memory
	WriteBaseRAM(uint16, uint8) // Write to 2k Base memory
	PrgRead(uint16) uint8       // Read PRG data
	PrgWrite(uint16, uint8)     // Write PRG data
	ChrRead(uint16) uint8       // Read CHR data
	ChrWrite(uint16, uint8)     // Write CHR data
	MirroringMode() uint8       // Which mirroring mode is tilemap data stored in
	HasSaveRAM() bool           // Whether or not the cartridge exposes Save RAM at 0x6000-0x7999
}

type baseMapper struct {
	id   uint16
	rom  *nesrom.ROM
	name string
	//The base amount of NES RAM (2k) will be accessed here.
	baseRAM []uint8
}

func newBaseMapper(id uint16, name string) *baseMapper {
	return &baseMapper{
		id:      id,
		name:    name,
		baseRAM: make([]uint8, NES_BASE_MEMORY),
	}
}

func (bm *baseMapper) ReadBaseRAM(addr uint16) uint8 {
	return bm.baseRAM[addr]
}

func (bm *baseMapper) WriteBaseRAM(addr uint16, val uint8) {
	bm.baseRAM[addr] = val
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

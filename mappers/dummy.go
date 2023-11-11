package mappers

import (
	"math"

	"github.com/bdwalton/gintendo/nesrom"
)

type dummyMapper struct {
	memory []uint8
	MM     uint8 // mirroring mode - tests can set as needed
}

func (dm *dummyMapper) ID() uint16 {
	return 0
}

func (dm *dummyMapper) Init(r *nesrom.ROM) {
	return
}

func (dm *dummyMapper) Name() string {
	return "dummy mapper"
}

func (dm *dummyMapper) ReadBaseRAM(addr uint16) uint8 {
	return dm.memory[addr]
}

func (dm *dummyMapper) WriteBaseRAM(addr uint16, val uint8) {
	dm.memory[addr] = val
}

func (dm *dummyMapper) PrgRead(addr uint16) uint8 {
	return dm.memory[addr]
}

func (dm *dummyMapper) PrgWrite(addr uint16, val uint8) {
	dm.memory[addr] = val
}

func (dm *dummyMapper) ChrRead(addr uint16) uint8 {
	return dm.memory[addr]
}

func (dm *dummyMapper) ChrWrite(addr uint16, val uint8) {
	dm.memory[addr] = val
}

func (dm *dummyMapper) MirroringMode() uint8 {
	return dm.MM
}

func (dm *dummyMapper) HasSaveRAM() bool {
	return true
}

func (dm *dummyMapper) LoadMem(start uint8, mem []uint8) {
	for i, m := range mem {
		dm.memory[int(start)+i] = m
	}
}

func (dm *dummyMapper) ClearMem() {
	dm.memory = make([]uint8, math.MaxUint16+1)
}

// For testing
var Dummy *dummyMapper = &dummyMapper{memory: make([]uint8, math.MaxUint16+1)}

package main

import (
	"flag"
	"log"
	"os"

	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/mos6502"
	"github.com/bdwalton/gintendo/nesrom"
)

var (
	romFile = flag.String("nes_rom", "", "Path to NES ROM to run.")
)

func main() {
	flag.Parse()

	rf, err := os.Open(*romFile)
	if err != nil {
		log.Fatalf("Couldn't open %q: %v", *romFile, err)
	}

	rom, err := nesrom.New(rf)
	if err != nil {
		log.Fatalf("Invalid ROM: %v", err)
	}

	m, err := mappers.Get(rom)
	if err != nil {
		log.Fatalf("Couldn't Get() mapper: %v", err)
	}

	g := mos6502.New(m)

	g.BIOS()
}

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bdwalton/gintendo/console"
	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/nesrom"
)

var (
	romFile = flag.String("nes_rom", "", "Path to NES ROM to run.")
	rawMode = flag.Bool("raw_mode", false, "If true, use the Dummy mapper with nes_rom loaded at 0x000A")
)

func main() {
	flag.Parse()

	var m mappers.Mapper

	if *rawMode {
		bin, err := os.ReadFile(*romFile)
		if err != nil {
			log.Fatalf("Invalid ROM: %v", err)
		}
		d := mappers.Dummy
		d.LoadMem(0x000A, bin)
		m = d
	} else {
		rom, err := nesrom.New(*romFile)
		if err != nil {
			log.Fatalf("Invalid ROM: %v", err)
		}
		m, err = mappers.Get(rom)
		if err != nil {
			log.Fatalf("Couldn't Get() mapper: %v", err)
		}
	}

	gintendo := console.New(m)
	gintendo.BIOS(context.Background())
}

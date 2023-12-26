package main

import (
	"flag"
	"log"
	"os"

	"github.com/bdwalton/gintendo/console"
	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/nesrom"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	romFile = flag.String("nes_rom", "", "Path to NES ROM to run.")
	nesMode = flag.Bool("nes_mode", true, "If false, use the Dummy mapper with nes_rom treated as a 64k binary loaded at 0x000A")
)

func main() {
	flag.Parse()

	var m mappers.Mapper
	mode := console.NES_MODE
	var gintendo *console.Bus

	if *nesMode {
		rom, err := nesrom.New(*romFile)
		if err != nil {
			log.Fatalf("Invalid ROM: %v", err)
		}
		m, err = mappers.Get(rom)
		if err != nil {
			log.Fatalf("Couldn't Get() mapper: %v", err)
		}
	} else {
		mode = console.REG_MODE
		m = mappers.Dummy
	}

	gintendo = console.New(m, mode)

	if !*nesMode {
		bin, err := os.ReadFile(*romFile)
		if err != nil {
			log.Fatalf("Invalid ROM: %v", err)
		}
		gintendo.LoadMem(0x000A, bin)
	}

	if err := ebiten.RunGame(gintendo); err != nil {
		log.Fatal(err)
	}
}

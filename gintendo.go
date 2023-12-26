package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bdwalton/gintendo/console"
	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/nesrom"
	"github.com/veandco/go-sdl2/sdl"
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

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatalf("Couldn't initialize SDL: %v", err)
	}

	window, err := sdl.CreateWindow("Gintendo", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 256, 240, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatalf("Couldn't create sdl window: %v", err)
	}
	defer window.Destroy()
	sdl.DisableScreenSaver()
	defer sdl.EnableScreenSaver()

	gintendo, err = console.New(m, mode, window)
	if err != nil {
		log.Fatalf("Couldn't create console object: %v", err)
	}

	if !*nesMode {
		bin, err := os.ReadFile(*romFile)
		if err != nil {
			log.Fatalf("Invalid ROM: %v", err)
		}
		gintendo.LoadMem(0x000A, bin)
	}

	gintendo.BIOS(context.Background())
}

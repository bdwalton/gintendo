package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bdwalton/gintendo/console"
	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/nesrom"
	"github.com/hajimehoshi/ebiten/v2"
)

var romFile = flag.String("nes_rom", "", "Path to NES ROM to run.")

func main() {
	flag.Parse()

	var m mappers.Mapper
	var gintendo *console.Bus

	rom, err := nesrom.New(*romFile)
	if err != nil {
		log.Fatalf("Invalid ROM: %v", err)
	}
	m, err = mappers.Get(rom)
	if err != nil {
		log.Fatalf("Couldn't Get() mapper: %v", err)
	}

	gintendo = console.New(m)

	go func() {
		if err := ebiten.RunGame(gintendo); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	gintendo.BIOS(context.Background())
}

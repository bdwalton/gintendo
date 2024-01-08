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

var romFile = flag.String("nes_rom", "", "Path to NES ROM to run.")

func main() {
	flag.Parse()

	rom, err := nesrom.New(*romFile)
	if err != nil {
		log.Fatalf("Invalid ROM: %v", err)
	}

	m, err := mappers.Get(rom)
	if err != nil {
		log.Fatalf("Couldn't Get() mapper: %v", err)
	}

	if err := ebiten.RunGame(console.New(m)); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

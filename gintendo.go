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

	rom, err := nesrom.New(*romFile)
	if err != nil {
		log.Fatalf("Invalid ROM: %v", err)
	}

	m, err := mappers.Get(rom)
	if err != nil {
		log.Fatalf("Couldn't Get() mapper: %v", err)
	}

	gintendo := console.New(m)

	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		gintendo.Run(ctx)
	}(ctx)

	if err := ebiten.RunGame(gintendo); err != nil {
		log.Fatal(err)

	}

	cancel()
	os.Exit(0)
}

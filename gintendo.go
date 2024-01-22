package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bdwalton/gintendo/console"
	"github.com/bdwalton/gintendo/mappers"
	"github.com/hajimehoshi/ebiten/v2"
)

var romFile = flag.String("nes_rom", "", "Path to NES ROM to run.")

func main() {
	flag.Parse()

	m, err := mappers.Load(*romFile)
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

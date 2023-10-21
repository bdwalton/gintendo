package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

	fmt.Println(rom)

	g := mos6502.New()
	fmt.Println(g)
}

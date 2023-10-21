package main

import (
	"flag"
	"fmt"
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

	m, ok := mappers.AllMappers[rom.MapperNum()]
	if !ok {
		log.Fatalf("Unimplemnted mapper id %d.", rom.MapperNum())
	}
	g := mos6502.New(m)
	fmt.Println(g)
}

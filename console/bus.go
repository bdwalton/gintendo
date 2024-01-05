package console

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/bdwalton/gintendo/mappers"
	"github.com/bdwalton/gintendo/mos6502"
	"github.com/bdwalton/gintendo/ppu"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	NES_BASE_MEMORY = 0x800 // 2KB built in RAM

	MAX_ADDRESS          = math.MaxUint16
	MEM_SIZE             = MAX_ADDRESS + 1
	MAX_NES_BASE_RAM     = 0x1FFF
	MAX_PPU_REG_MIRRORED = 0x3FFF
	MAX_IO_REG           = 0x4020
	MAX_SRAM             = 0x6000
)

const (
	OAMDMA = 0x4014 // Triggers DMA from CPU memory to DMA
)

type Bus struct {
	cpu    *mos6502.CPU
	ppu    *ppu.PPU
	mapper mappers.Mapper
	ram    []uint8
	ticks  uint64
}

func New(m mappers.Mapper) *Bus {
	bus := &Bus{mapper: m, ram: make([]uint8, NES_BASE_MEMORY)}

	bus.cpu = mos6502.New(bus)
	bus.ppu = ppu.New(bus)

	w, h := bus.ppu.GetResolution()
	ebiten.SetWindowSize(w*2, h*2) // Start with 2x the screen size
	ebiten.SetWindowTitle("Gintendo")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	return bus
}

func (b *Bus) MirrorMode() uint8 {
	return b.mapper.MirroringMode()
}

// Layout returns the constant resolution of the NES and is part of
// the ebiten.Game interface. By returning constants here, we will
// force ebiten to scale the display when the window size changes.
func (b *Bus) Layout(w, h int) (int, int) {
	return b.ppu.GetResolution()
}

// Draw updates the displayed ebiten window with the current state of
// the PPU.
func (b *Bus) Draw(screen *ebiten.Image) {
	px := b.ppu.GetPixels()
	rect := px.Bounds()
	dx, dy := rect.Dx(), rect.Dy()

	for x := 0; x < dx; x++ {
		for y := 0; y < dy; y++ {
			screen.Set(x, y, px.At(x, y))
		}
	}
}

// Update is called by ebiten roughly every 1/60s and will be our
// driver for the emulation.
func (b *Bus) Update() error {
	// We do work in a different goroutine and don't need ebiten
	// to drive this. We have to be implemented and called though
	// as it's part of the required interface.
	return nil
}

// TriggerNMI is used by the PPU to signal the CPU that it is in vblank.
func (b *Bus) TriggerNMI() {
	b.cpu.TriggerNMI()
}

// ChrRead is used by the PPU to access CHR-ROM in the loaded Mapper
func (b *Bus) ChrRead(addr uint16) uint8 {
	return b.mapper.ChrRead(addr)
}

func (b *Bus) Read(addr uint16) uint8 {
	// https://www.nesdev.org/wiki/CPU_memory_map
	switch {
	case addr <= MAX_NES_BASE_RAM:
		// 0x800-0x1FFF mirrors 0x0000-0x07FF
		return b.ram[addr&0x7FF]
	case addr <= MAX_PPU_REG_MIRRORED:
		// PPU registers are mirrored between 0x2000 and 0x4000
		return b.ppu.ReadReg(addr & 0x2007)
	case addr < MAX_IO_REG:
		// handle joysticks
		return 0
	case addr <= MAX_SRAM:
		return 0
	case addr <= MAX_ADDRESS:
		return b.mapper.PrgRead(addr)
	}

	panic("should never happen") // hah, prod crashes await!
}

func (b *Bus) ClearMem() {
	b.ram = make([]uint8, len(b.ram))
}

func (b *Bus) Write(addr uint16, val uint8) {
	// https://www.nesdev.org/wiki/CPU_memory_map
	switch {
	case addr <= MAX_NES_BASE_RAM:
		// 0x800-0x1FFF mirrors 0x0000-0x07FF
		b.ram[addr&0x07FF] = val
	case addr <= MAX_PPU_REG_MIRRORED:
		// PPU registers are mirrored between 0x2000 and 0x4000
		b.ppu.WriteReg(addr&0x2007, val)
	case addr < MAX_IO_REG:
		// Handle Joysticks, APU and PPU DMA
		switch addr {
		case OAMDMA:
			// TODO: Smooth this out across PPU cycles
			base := uint16(val) << 8
			for addr := base; addr < base+256; addr++ {
				b.ppu.WriteReg(ppu.OAMDATA, b.Read(addr))
			}
			b.cpu.AddDMACycles()
		}
	case addr <= MAX_SRAM:
		// nothing for now
	case addr <= MAX_ADDRESS:
		b.mapper.PrgWrite(addr, val)
	}
}

func readAddress(prompt string) uint16 {
	var a uint16
	fmt.Printf(prompt)
	fmt.Scanf("%04x\n", &a)
	return a
}

func (b *Bus) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			b.ppu.Tick()
			if b.ticks%3 == 0 {
				b.cpu.Tick()
			}
			b.ticks += 1
		}
	}
}

func (b *Bus) BIOS(ctx context.Context) {
	sigQuit := make(chan os.Signal, 1)
	signal.Notify(sigQuit, syscall.SIGINT, syscall.SIGTERM)

	breaks := make(map[uint16]struct{})

	for {
		fmt.Printf("%s\n\n", b.cpu)
		fmt.Println("(B)reak - add breakpoint")
		fmt.Println("(C)lear - cleear breakpoints")
		fmt.Println("(R)un - run to completion")
		fmt.Println("(S)step - step the cpu one instruction")
		fmt.Println("R(e)set - hit the reset button")
		fmt.Println("(M)memory - select a memory range to display")
		fmt.Println("S(t)ack - show last 3 items on the stack")
		fmt.Println("(I)instruction - show instruction memory locations")
		fmt.Println("(P)C - set program counter")
		fmt.Println("PP(U) - show PPU status")
		fmt.Println("(Q)uit - shutdown the gintentdo")
		fmt.Printf("Choice: ")

		var in rune
		fmt.Scanf("%c\n", &in)

		switch in {
		case 'b', 'B':
			breaks[readAddress("Breakpoint (eg: ff15): ")] = struct{}{}
		case 'c', 'C':
			breaks = make(map[uint16]struct{})
		case 'p', 'P':
			b.cpu.SetPC(readAddress("Set PC to what address (eg: 0400)?: "))
		case 'q', 'Q':
			return
		case 'r', 'R':
			cctx, cancel := context.WithCancel(ctx)
			go func(ctx context.Context) {
				for {
					select {
					case <-sigQuit:
						cancel()
					case <-ctx.Done():
						return
					}
				}
			}(cctx)

			b.Run(cctx)
		case 's', 'S':
			c := b.cpu.Step() * 3
			for i := 0; i < c; i++ {
				b.ppu.Tick()
			}
		case 't', 'T':
			fmt.Println()
			i := 0
			for {
				m := b.cpu.StackAddr() + uint16(i)
				fmt.Printf("0x%04x: 0x%02x ", m, b.Read(m))
				if m == 0x01ff || i == 2 {
					break
				}
				i += 1
			}
			fmt.Printf("\n\n")
		case 'i', 'I':
			fmt.Printf("\n%s\n\n", b.cpu.Inst())
		case 'u', 'U':
			fmt.Println(b.ppu)
		case 'e', 'E':
			b.cpu.Reset()
		case 'm', 'M':
			fmt.Println()
			low := readAddress("Low address (eg f00d): ")
			high := readAddress("High address (eg beef): ")
			fmt.Println()

			x := 1
			i := low
			for {
				fmt.Printf("0x%04x: 0x%02x ", i, b.Read(i))
				if x%5 == 0 {
					fmt.Println()
				}
				if i == high || i == math.MaxUint16 {
					break
				}
				x += 1
				i += 1
			}
			fmt.Printf("\n\n")
		}
	}
}

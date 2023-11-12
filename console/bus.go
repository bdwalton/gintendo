package console

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/bdwalton/gintendo/mappers"
)

type Bus struct {
	cpu    *CPU
	ppu    *PPU
	mapper mappers.Mapper
}

func New(m mappers.Mapper) *Bus {
	bus := &Bus{mapper: m}
	bus.cpu = newCPU(bus, m)
	bus.ppu = newPPU(bus, m)

	return bus
}

func (b *Bus) Read(addr uint16) uint8 {
	// https://www.nesdev.org/wiki/CPU_memory_map
	switch {
	case addr <= 0x1FFF:
		// 0x800-0x1FFF mirrors 0x0000-0x07FF
		return b.mapper.ReadBaseRAM(addr % 0x800)
	case addr < MAX_IO_REG_MIRRORED:
		// PPU registers are mirrored between 0x2000 and 0x4000
		return b.ppu.ReadReg(0x2000 + ((addr - 0x2000) % 0x8))
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

func (b *Bus) Write(addr uint16, val uint8) {
	// https://www.nesdev.org/wiki/CPU_memory_map
	switch {
	case addr <= 0x1FFF:
		// 0x800-0x1FFF mirrors 0x0000-0x07FF
		b.mapper.WriteBaseRAM(addr%0x800, val)
	case addr < MAX_IO_REG_MIRRORED:
		// PPU registers are mirrored between 0x2000 and 0x4000
		b.ppu.WriteReg(0x2000+((addr-0x2000)%0x8), val)
	case addr < MAX_IO_REG:
		// handle joysticks
	case addr <= MAX_SRAM:
		// nothing for now
	case addr <= MAX_ADDRESS:
		b.mapper.PrgWrite(addr, val)
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
			b.cpu.pc = readAddress("Set PC to what address (eg: 0400)?: ")
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
			b.cpu.Run(cctx, breaks)
		case 's', 'S':
			b.cpu.step()
		case 't', 'T':
			fmt.Println()
			i := 0
			for {
				m := b.cpu.getStackAddr() + uint16(i)
				fmt.Printf("0x%04x: 0x%02x ", m, b.cpu.read(m))
				if m == 0x00ff || i == 2 {
					break
				}
				i += 1
			}
			fmt.Printf("\n\n")
		case 'i', 'I':
			fmt.Println()
			op := opcodes[b.cpu.read(b.cpu.pc)]
			for i := 0; i < int(op.bytes); i++ {
				m := b.cpu.pc + uint16(i)
				fmt.Printf("0x%04x: 0x%02x ", m, b.cpu.read(m))
			}
			fmt.Printf("\n\n")
		case 'e', 'E':
			b.cpu.reset()
		case 'm', 'M':
			fmt.Println()
			low := readAddress("Low address (eg f00d): ")
			high := readAddress("High address (eg beef): ")
			fmt.Println()

			x := 1
			i := low
			for {
				fmt.Printf("0x%04x: 0x%02x ", i, b.cpu.read(i))
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

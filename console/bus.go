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

type bus struct {
	cpu *CPU
	ppu *PPU
}

func New(m mappers.Mapper) *bus {
	b := &bus{}
	b.cpu = newCPU(b, m)
	b.ppu = newPPU(b, m)

	return b
}

func (b *bus) BIOS(ctx context.Context) {
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
		fmt.Println("(Q)uit - shutdown the gintentdo")
		fmt.Printf("Choice: ")

		var in rune
		fmt.Scanf("%c\n", &in)

		switch in {
		case 'b', 'B':
			breaks[readAddress("Breakpoint (eg: ff15): ")] = struct{}{}
		case 'c', 'C':
			breaks = make(map[uint16]struct{})
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
				fmt.Printf("0x%04x: 0x%02x ", m, b.cpu.mem.read(m))
				if m == 0x00ff || i == 2 {
					break
				}
				i += 1
			}
			fmt.Printf("\n\n")
		case 'i', 'I':
			fmt.Println()
			op := opcodes[b.cpu.mem.read(b.cpu.pc)]
			for i := 0; i < int(op.bytes); i++ {
				m := b.cpu.pc + uint16(i)
				fmt.Printf("0x%04x: 0x%02x ", m, b.cpu.mem.read(m))
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
				fmt.Printf("0x%04x: 0x%02x ", i, b.cpu.mem.read(i))
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

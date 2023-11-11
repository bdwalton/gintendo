// package nesrom implements support for the NES (iNES, NES2) ROM
// format. https://www.nesdev.org/wiki/INES
package nesrom

import (
	"fmt"
	"os"
	"strings"
)

type PlayChoicePROM struct {
	Data       [16]byte
	CounterOut [16]byte
}

type ROM struct {
	path      string
	h         *header
	trainer   []byte          // if present
	prg       []byte          // 16384 * x bytes; x from header
	chr       []byte          // 8192 * y bytes; y from header
	pcInstRom []byte          // if present
	pcPROM    *PlayChoicePROM // if present; often missing - see PC10 ROM-Images
}

const (
	TRAINER_SIZE   = 512
	PRG_BLOCK_SIZE = 16384
	CHR_BLOCK_SIZE = 8192
	PC_INST_SIZE   = 8192
	PC_PROM_SIZE   = 32
)

func New(path string) (*ROM, error) {
	rf, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open ROM file %q: %w", path, err)
	}

	hbytes := make([]byte, 16)
	n, err := rf.Read(hbytes)
	if n != 16 || err != nil {
		return nil, fmt.Errorf("couldn't read header: %w", err)
	}

	i, err := &ROM{path: path, h: parseHeader(hbytes)}, nil
	if err != nil {
		return nil, fmt.Errorf("error parsing header %w", err)
	}
	if i.h.hasTrainer() {
		i.trainer = make([]byte, TRAINER_SIZE)
		if n, err := rf.Read(i.trainer); n != TRAINER_SIZE || err != nil {
			return nil, fmt.Errorf("error reading trainer data: %w", err)
		}

	}

	s := PRG_BLOCK_SIZE * int(i.h.prgSize)
	i.prg = make([]byte, s)
	if n, err := rf.Read(i.prg); n != s || err != nil {
		return nil, fmt.Errorf("error reading PRG ROM (read %d, wanted %d): %w", n, s, err)
	}

	s = CHR_BLOCK_SIZE * int(i.h.chrSize)
	i.chr = make([]byte, s)
	if n, err := rf.Read(i.chr); n != s || err != nil {
		return nil, fmt.Errorf("error reading CHR ROM (read %d, wanted %d): %w", n, s, err)
	}

	if i.h.hasPlayChoice() {
		i.pcInstRom = make([]byte, PC_INST_SIZE)
		if n, err := rf.Read(i.pcInstRom); n != PC_INST_SIZE || err != nil {
			return nil, fmt.Errorf("error reading PlayChoice INSt ROM (n=%d; wanted %d): %w", n, PC_INST_SIZE, err)
		}

		// Some old ROMs may not have this, so bailing might
		// be bad. But these should be rare, so we'll do the
		// technically correct thing for now.
		pcprom := make([]byte, PC_PROM_SIZE)
		if n, err := rf.Read(pcprom); n != PC_PROM_SIZE || err != nil {
			return nil, fmt.Errorf("error reading PlayChoice PROM (n=%d, wanted %d): %w", n, PC_PROM_SIZE, err)
		}
	}

	return i, nil
}

func (r *ROM) NumPrgBlocks() uint8 {
	return r.h.prgSize
}

func (r *ROM) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", r.h))
	if r.h.hasTrainer() {
		sb.WriteString(fmt.Sprintf("Trainer: %v\n", r.trainer))
	}

	sb.WriteString(fmt.Sprintf("PRG: %v\n", r.prg))
	sb.WriteString(fmt.Sprintf("CHR: %v\n", r.chr))

	return sb.String()
}

func (r *ROM) PrgRead(addr uint16) uint8 {
	return r.prg[addr]
}

func (r *ROM) PrgWrite(addr uint16, val uint8) {
	r.prg[addr] = val
}

func (r *ROM) ChrRead(addr uint16) uint8 {
	return r.chr[addr]
}

func (r *ROM) ChrWrite(addr uint16, val uint8) {
	r.chr[addr] = val
}

func (r *ROM) MapperNum() uint16 {
	return r.h.mapperNum()
}

func (r *ROM) MirroringMode() uint8 {
	return r.h.mirroringMode()
}

func (r *ROM) HasSaveRAM() bool {
	return r.h.hasPrgRAM()
}

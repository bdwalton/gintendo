// package nesFormat implements support for the NES (iNES) ROM format
// https://www.nesdev.org/wiki/INES
package nesFormat

import (
	"fmt"
	"io"
)

type PlayChoicePROM struct {
	Data       [16]byte
	CounterOut [16]byte
}

type ROM struct {
	h         *Header
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
)

func New(inesData io.Reader) (*ROM, error) {
	hbytes := make([]byte, 16)
	n, err := inesData.Read(hbytes)
	if n != 16 || err != nil {
		return nil, fmt.Errorf("couldn't read header: %w", err)
	}

	i, err := &ROM{h: parseHeader(hbytes)}, nil
	if err != nil {
		return nil, fmt.Errorf("error parsing header %w", err)
	}
	if i.h.HasTrainer() {
		i.trainer = make([]byte, TRAINER_SIZE)
		if n, err := inesData.Read(i.trainer); n != TRAINER_SIZE || err != nil {
			return nil, fmt.Errorf("error reading trainer data: %w", err)
		}

	}

	s := PRG_BLOCK_SIZE * int(i.h.PrgSize())
	i.prg = make([]byte, s)
	if n, err := inesData.Read(i.prg); n != s || err != nil {
		return nil, fmt.Errorf("error reading PRG ROM (read %d, wanted %d): %w", n, s, err)
	}

	s = CHR_BLOCK_SIZE * int(i.h.ChrSize())
	i.chr = make([]byte, s)
	if n, err := inesData.Read(i.chr); n != s || err != nil {
		return nil, fmt.Errorf("error reading CHR ROM (read %d, wanted %d): %w", n, s, err)
	}

	return i, nil
}

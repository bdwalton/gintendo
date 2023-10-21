package nesrom

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	f, err := os.Open("../testdata/ram_after_reset.nes")
	if err != nil {
		t.Errorf("couldn't open testdata file: %v", err)
	}

	if _, err := New(f); err != nil {
		t.Errorf("couldn't parse testdata file: %v", err)
	}
}

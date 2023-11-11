package nesrom

import (
	"testing"
)

func TestNew(t *testing.T) {
	if _, err := New("../testdata/ram_after_reset.nes"); err != nil {
		t.Errorf("couldn't parse testdata file: %v", err)
	}
}

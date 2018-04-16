package paeudo

import (
	"fmt"
	"testing"
)

func TestPaeudo1(t *testing.T) {
	err := PaeudoShell("/home/jason/go")
	if err != nil {
		t.Error(fmt.Sprintf("error occured: %s", err))
	}
}

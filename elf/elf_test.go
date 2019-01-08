package elf

import (
	"flag"
	"testing"
)

var name string

func init() {
	flag.StringVar(&name, "name", "", "the name for ld.so")
	flag.Parse()
}

func TestELF1(t *testing.T) {
	t.Log(name)
	err := Patchldso(name, fmt.Sprintf("%s.patch", name))
	if err != nil {
		t.Error(err)
	}
}

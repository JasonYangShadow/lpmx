package elf

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestELF1(t *testing.T) {
	err := Patchldso("/tmp/ld-2.23.so")
	if err != nil {
		t.Error(err)
	}
	content, _ := ioutil.ReadFile("/tmp/ld-2.23.so.patch")
	etc := []byte("\x00/etc")
	if bytes.Contains(content, etc) {
		t.Error("patch failed")
	}
}

package utils

import (
	"fmt"
	"testing"
)

func TestUtils1(t *testing.T) {
	file := "/home/jason/.spacemacs"
	ret := FileExist(file)
	fmt.Printf("%t", ret)
}

func TestUtils2(t *testing.T) {
	src := "/tmp/log"
	dst := "/tmp/log.bak"
	CopyFile(src, dst)
}

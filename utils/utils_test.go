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

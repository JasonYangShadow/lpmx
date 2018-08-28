package utils

import (
	"fmt"
	"testing"
)

func TestUtilsUntar(t *testing.T) {
	//t.Skip("skip test")
	err := Untar("/tmp/8e3ba11ec2a2b39ab372c60c16b421536e50e5ce64a0bc81765c2e38381bcff6", "/tmp/alpine")
	if err != nil {
		t.Error(err)
	}
}

func TestUtils1(t *testing.T) {
	t.Skip("skip test")
	file := "/home/jason/.spacemacs"
	ret := FileExist(file)
	fmt.Printf("%t", ret)
}

func TestUtils2(t *testing.T) {
	t.Skip("skip test")
	src := "/tmp/log"
	dst := "/tmp/log.bak"
	val, err := CopyFile(src, dst)
	if val {
		t.Log("successfully copied")
	} else {
		t.Error(err)
	}
}

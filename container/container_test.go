package container

import (
	"fmt"
	"testing"
)

func TestContainer1(t *testing.T) {
	con, _ := CreateContainer(".", "test")
	fmt.Println(Walkfs(con))
}

func TestContainer2(t *testing.T) {
	fmt.Println(Command("ls", "-al"))
}

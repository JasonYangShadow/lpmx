package container

import (
	"fmt"
	"testing"
)

func TestContainer1(t *testing.T) {
	fmt.Println(Command("ls", "-al"))
}

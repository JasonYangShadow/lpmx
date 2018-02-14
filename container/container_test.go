package container

import (
	"fmt"
	"testing"
)

func TestContainer1(t *testing.T) {
	con, err := CreateContainer("./", "test")
	fmt.Printf(con)
}

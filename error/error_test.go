package error

import (
	"fmt"
	"testing"
)

func TestErr1(t *testing.T) {
  err := ErrNew(ErrNil,"msg")
	fmt.Println(err.Error())
}

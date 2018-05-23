package error

import (
	"fmt"
	"testing"
)

func TestErr1(t *testing.T) {
	err := ErrNew(ErrNil, "msg")
	err.AddMsg("msg1")
	fmt.Println(err.Error())
}

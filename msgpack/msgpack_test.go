package msgpack

import (
	. "github.com/jasonyangshadow/lpmx/container"
	"testing"
)

func TestStructMarshal1(t testing.T) {
	var mem MemContainers
	mem.ContainersMap["k1"] = "val1"
	mem.ContainersMap["k2"] = "val2"

	d, err := StructMarshal(mem)
	if err != nil {
		t.Error(err)
	}
}

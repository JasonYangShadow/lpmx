package memcache

import (
	"testing"
)

func TestMem1(t *testing.T) {
	mem, err := MInitServer()
	t.Log(mem.ClientInst)
	if err == nil {
		value, _ := mem.MGetStrValue("ZIKqoRpKaZ:bash")
		t.Log(value)
	} else {
		t.Error(err)
	}
}

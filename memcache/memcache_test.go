package memcache

import (
	"testing"
)

func TestMem1(t *testing.T) {
	mem, err := MInitServer()
	t.Log(mem.ClientInst)
	if err == nil {
		value, _ := mem.MGetStrValue("aJ2Q0pV6bL:bash")
		t.Log(value)
	} else {
		t.Error(err)
	}
}

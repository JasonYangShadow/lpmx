package memcache

import (
	"testing"
)

func TestMem1(t *testing.T) {
	mem, err := MInitServer()
	t.Log(mem.ClientInst)
	if err == nil {
		err = mem.MSetStrValue("8exMB69sW9:bash:allow", "all")
		if err != nil {
			t.Error(err)
		}
		value, _ := mem.MGetStrValue("8exMB69sW9:bash:allow")
		t.Log(value)
	} else {
		t.Error(err)
	}
}

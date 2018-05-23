package memcache

import (
	"testing"
)

func TestMem1(t *testing.T) {
	mem, err := MInitServers("/home/jason/mem.pid")
	if err != nil {
		t.Error(err)
	}
	err = mem.MSetStrValue("hh", "all")
	if err != nil {
		t.Error(err)
	}
	str, _ := mem.MGetStrValue("hh")
	t.Log(str)
}

func TestMem2(t *testing.T) {
	mem, err := MInitServer()
	t.Log(mem.ClientInst)
	if err == nil {
		mem.MSetStrValue("map:vjD4vhWkGB:vim", "all")
		value, _ := mem.MGetStrValue("map:vjD4vhWkGB:vim")
		t.Log(value)
	} else {
		t.Error(err)
	}
}

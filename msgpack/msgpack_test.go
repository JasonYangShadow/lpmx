package msgpack

import (
	. "github.com/jasonyangshadow/lpmx/container"
	"testing"
)

func TestStructMarshal1(t *testing.T) {
	var mem MemContainers
	var con Container
	con.ContainerName = "container"
	con.CreateUser = "jason"

	setmap := make(map[string]interface{})
	setmap["k1"] = "val1"
	con.SettingConf = setmap

	containermap := make(map[string]*Container)
	containermap["con1"] = &con

	mem.ContainersMap = containermap
	mem.RootDir = "root"

	d, err := StructMarshal(mem)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", string(d))

	mem_new := new(MemContainers)
	err = StructUnmarshal(d, mem_new)
	if err != nil {
		t.Error(err)
	}
	t.Log(mem_new)
	t.Log((*mem_new).RootDir)
	con_map := (*mem_new).ContainersMap
	t.Log(con_map["con1"])
	con_pointer := con_map["con1"]
	t.Log((*con_pointer).SettingConf)
}

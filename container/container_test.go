package container

import (
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"testing"
)

var mem struct {
	p_mem *MemContainers
	p_con *Container
}

func TestContainerInit(t *testing.T) {
	confs := []string{".", "/tmp/lpmx_root"}
	val, _ := GetMap("setting", confs)
	var err *Error
	mem.p_mem, err = Init(confs)
	mem.p_mem.SettingConf = val
	if err == nil {
		t.Log(mem.p_mem)
	} else {
		t.Error(err)
	}
}

func TestContainerCreateContainer(t *testing.T) {
	var err *Error
	mem.p_con, err = mem.p_mem.CreateContainer("/tmp/lpmx_test", "test")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(mem.p_con)
	}
}

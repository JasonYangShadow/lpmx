package container

import (
	//	. "github.com/jasonyangshadow/lpmx/error"
	//. "github.com/jasonyangshadow/lpmx/yaml"
	"testing"
)

var mem struct {
	p_mem *MemContainers
	p_con *Container
}

/**
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
**/

func TestContainer(t *testing.T) {
	dir := "/tmp/lpmx_test"
	name := "test"
	confs := []string{"/tmp/lpmx_root"}
	mem, ierr := Init(confs)
	if ierr == nil {
		con, cerr := mem.CreateContainer(dir, name)
		t.Log(con)
		if cerr == nil {
			t.Log(mem)
			_, rerr := mem.RunContainer(con.Id)
			if rerr != nil {
				t.Error(rerr)
			}
		} else {
			t.Error(cerr)
		}
	} else {
		t.Error(ierr)
	}
}

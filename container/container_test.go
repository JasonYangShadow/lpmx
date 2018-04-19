package container

import (
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"testing"
)

var mem struct {
	pointer *MemContainers
}

func TestContainerInit(t *testing.T) {
	confs := []string{".", "/tmp/lpmx_root"}
	val, _ := GetMap("setting", confs)
	var err *Error
	mem.pointer, err = Init(confs)
	mem.pointer.SettingConf = val
	if err == nil {
		t.Log(mem.pointer)
	} else {
		t.Error(err)
	}
}

func TestContainerCreate(t *testing.T) {
	t.Log(mem.pointer)
	con, err := createContainer(mem.pointer.RootDir, "/tmp/lpmx_test", "test")
	if err == nil {
		t.Log(con)
		files, err := WalkContainerRoot(con)
		t.Log(files)
		if err == nil {
			for _, file := range files {
				val, err := ElfRPath(con.ElfPatcherPath, con.SettingConf["libpath"].(string), file)
				if err == nil {
					t.Log(val)
				} else {
					t.Error(err)
				}
			}
		}
	}
	/**con, err := mem.pointer.CreateContainer("/tmp/lpmx_test", "test")
	if err == nil {
		t.Log(con)
	} else {
		t.Error(err)
	}**/
}

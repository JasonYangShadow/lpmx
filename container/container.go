package container

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"os"
	"path/filepath"
	"strings"
)

const (
	RUNNING = iota
	PAUSE
	STOPPED

	MAX_CONTAINER_COUNT = 1024
)

var (
	AvailableContainerIds = [MAX_CONTAINER_COUNT]int8{0}
)

type MemContainers struct {
	ContainersMap map[string]*Container
	RootDir       string
	SettingConf   map[string]interface{}
}

type Container struct {
	Id                  string
	RootPath            string
	Status              int8
	LogPath             string
	ElfPatcherPath      string
	FakechrootPath      string
	SettingConfPath     string
	SettingConf         map[string]interface{}
	StartTime           string
	ImageName           string
	ContainerName       string
	CreateUser          string
	MemcachedServerList string
	ShmFiles            string
	IpcFiles            string
}

func findAvailableId() (int, *Error) {
	for i := 0; i < MAX_CONTAINER_COUNT; i++ {
		if AvailableContainerIds[i] == 0 {
			AvailableContainerIds[i] = 1
			return i, nil
		} else {
			continue
		}
	}
	cerr := ErrNew(ErrFull, "No available container id could be generated")
	return -1, &cerr
}

func createContainer(dir string, name string) (*Container, *Error) {
	id, err := findAvailableId()
	if err == nil {
		var con Container
		con.Id = fmt.Sprintf("container-%d", id)
		for strings.HasSuffix(dir, "/") {
			dir = strings.TrimSuffix(dir, "/")
		}
		con.RootPath = fmt.Sprintf("%s/%s/instance", dir, con.Id)
		con.Status = STOPPED
		con.LogPath = fmt.Sprintf("%s/%s/log", dir, con.Id)
		con.ElfPatcherPath = fmt.Sprintf("%s/%s/elf/", dir, con.Id)
		con.FakechrootPath = fmt.Sprintf("%s/%s/fakechroot/", dir, con.Id)
		con.SettingConfPath = fmt.Sprintf("%s/%s/settings/", dir, con.Id)
		con.SettingConf, _ = GetMap("setting.yml", []string{con.SettingConfPath})
		con.ImageName = name
		return &con, nil
	}
	return nil, err
}

func walkfs(dir string) ([]string, *Error) {
	fileList := []string{}

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		ftype, err := FileType(path)
		if err != nil {
			return err
		}
		if ftype == TYPE_REGULAR {
			_, err := FilePermission(path, PERM_EXE)
			if err != nil {
				return nil
			}
			fileList = append(fileList, path)
		}
		return nil
	})
	cerr := ErrNew(err, fmt.Sprintf("walkfs: %s error", dir))
	return fileList, &cerr
}

func Init(conf []string) (*MemContainers, *Error) {
	var cons MemContainers
	val, err := GetMap("setting.yml", conf)
	cons.SettingConf = val
	if err == nil {
		cons.RootDir = cons.SettingConf["RootDir"].(string)
		_, err := MakeDir(cons.RootDir)
		if err != nil {
			return nil, err
		}
		return &cons, nil
	}
	return nil, err
}

func (mem *MemContainers) CreateContainer(dir string, name string) (*Container, *Error) {
	con, err := createContainer(dir, name)
	if err == nil {
		files, err := WalkContainerRoot(con)
		if err == nil {
			for _, file := range files {
				if val, err := ElfRPath(con.ElfPatcherPath, con.SettingConf["libpath"].(string), file); val || err != nil {
					return con, err
				}
			}
		} else {
			return con, err
		}
		mem.ContainersMap[con.Id] = con
	}
	return nil, err
}

func (mem *MemContainers) RunContainer(id string) (*Container, *Error) {
	if con, ok := mem.ContainersMap[id]; ok {
		con.Status = RUNNING
		err := ContainerPaeudoShell(con.FakechrootPath, con.RootPath, con.ContainerName)
		if err == nil {
			return con, nil
		}
	}
	err := ErrNew(ErrNExist, fmt.Sprintf("Container ID: %s doesn't exist, please create it firstly", id))
	return nil, &err
}

func (mem *MemContainers) DestroyContainer(id string) *Error {
	if _, ok := mem.ContainersMap[id]; ok {
		delete(mem.ContainersMap, id)
		return nil
	} else {
		err := ErrNew(ErrNExist, fmt.Sprintf("Container ID: %s doesn't exist, please create it firstly", id))
		return &err
	}
}

func WalkContainerRoot(con *Container) ([]string, *Error) {
	return walkfs(con.RootPath)
}

func WalkSpecificDir(dir string) ([]string, *Error) {
	return walkfs(dir)
}

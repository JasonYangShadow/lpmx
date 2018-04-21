package container

import (
	"bufio"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/queue"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"github.com/spf13/viper"
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
	ELFOP                 = []string{"add_needed", "remove_needed", "add_rpath", "remove_rpath"}
)

type MemContainers struct {
	ContainersMap map[string]*Container
	RootDir       string
	SettingConf   map[string]interface{}
	V             *viper.Viper
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
	V                   *viper.Viper
	Qchan               chan Queue
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

func createSysFolders(con *Container, sysroot string, dir string, name string) *Error {
	_, err := MakeDir(fmt.Sprintf("%s/%s", sysroot, con.Id))
	if err != nil {
		return err
	}
	con.RootPath = dir
	con.Status = STOPPED
	con.LogPath = fmt.Sprintf("%s/%s/log", sysroot, con.Id)
	con.ElfPatcherPath = fmt.Sprintf("%s/%s/elf", sysroot, con.Id)
	con.FakechrootPath = fmt.Sprintf("%s/%s/fakechroot", sysroot, con.Id)
	con.SettingConfPath = fmt.Sprintf("%s/%s/settings", sysroot, con.Id)
	paths := []string{con.SettingConfPath}
	con.ContainerName = name
	con.Qchan = InitQueue(DEFAULT_SIZE)
	con.CreateUser, err = Command("whoami")
	if err != nil {
		return err
	}
	con.V, con.SettingConf, err = MultiGetMap("setting", paths)
	if err != nil {
		return err
	}
	_, err = MakeDir(con.LogPath)
	if err != nil {
		return err
	}
	_, err = MakeDir(con.ElfPatcherPath)
	if err != nil {
		return err
	}
	_, err = MakeDir(con.FakechrootPath)
	if err != nil {
		return err
	}
	_, err = MakeDir(con.SettingConfPath)
	if err != nil {
		return err
	}
	return nil
}

func createContainer(sysroot string, dir string, name string) (*Container, *Error) {
	id, err := findAvailableId()
	if err == nil {
		var con Container
		con.Id = fmt.Sprintf("container-%d", id)
		for strings.HasSuffix(dir, "/") {
			dir = strings.TrimSuffix(dir, "/")
		}
		err := createSysFolders(&con, sysroot, dir, name)
		if err != nil {
			return nil, err
		}
		return &con, nil
	}
	return nil, err
}

func walkfs(dir string) ([]string, *Error) {
	fileList := []string{}

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ftype, ferr := FileType(path)
		if ferr != nil {
			return ferr
		}
		if ftype == TYPE_REGULAR {
			ok, err := FilePermission(path, PERM_EXE)
			if err != nil {
				return err
			}
			if ok {
				fileList = append(fileList, path)
			}
		}
		return nil
	})
	if err != nil {
		cerr := ErrNew(err, "walkfs encountered error")
		return nil, &cerr
	}
	return fileList, nil
}

func Init(conf []string) (*MemContainers, *Error) {
	var cons MemContainers
	var err *Error
	cons.V, cons.SettingConf, err = MultiGetMap("setting", conf)
	cons.ContainersMap = make(map[string]*Container)
	if err == nil {
		cons.RootDir = cons.SettingConf["rootdir"].(string)
		_, err := MakeDir(cons.RootDir)
		if err != nil {
			return nil, err
		}
		return &cons, nil
	}
	return nil, err
}

func (con *Container) refreshElf(key string, value []string, prog string) *Error {
	switch key {
	case "add_rpath":
		{
			rpath := ""
			for _, path := range value {
				rpath = fmt.Sprintf("%s:%s", path, rpath)
			}
			if rpath != "" {
				rpath = strings.TrimSuffix(rpath, ":")
			}
			if val, err := ElfRPath(con.ElfPatcherPath, rpath, prog); val || err != nil {
				return err
			}
		}
	case "remove_rpath":
		{
			for _, path := range value {
				if val, err := ElfRemoveRPath(con.ElfPatcherPath, path, prog); val || err != nil {
					return err
				}
			}
		}
	case "add_needed":
		{
			if val, err := ElfAddNeeded(con.ElfPatcherPath, value, prog); val || err != nil {
				return err
			}
		}
	case "remove_needed":
		{
			if val, err := ElfRemoveNeeded(con.ElfPatcherPath, value, prog); val || err != nil {
				return err
			}
		}
	default:
		{
			cerr := ErrNew(ErrNExist, fmt.Sprintf("key: %s doesn't exist", key))
			return &cerr
		}
	}
	return nil
}

func (con *Container) ContainerPaeudoShell() *Error {
	if FolderExist(con.RootPath) {
		fmt.Println(fmt.Sprintf("%s@%s>>", con.CreateUser, con.ContainerName))
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "exit" {
				break
			}
			cmds := strings.Fields(text)
			env := make(map[string]string)
			env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so", con.FakechrootPath)
			val, err := CommandEnv(cmds[0], env, cmds[1:]...)
			if err == nil {
				fmt.Println(val)
			} else {
				fmt.Println(err)
			}
			fmt.Println(fmt.Sprintf("%s@%s>>", con.CreateUser, con.ContainerName))
		}
		return nil
	}
	cerr := ErrNew(ErrNExist, fmt.Sprintf("can't locate container root folder %s", con.RootPath))
	return &cerr
}

func (mem *MemContainers) CreateContainer(dir string, name string) (*Container, *Error) {
	con, err := createContainer(mem.RootDir, dir, name)
	if err == nil {
		mem.ContainersMap[con.Id] = con
		bineries, _, err := WalkContainerRoot(con)
		if err == nil {
			for _, binery := range bineries {
				for _, op := range ELFOP {
					if val, ok := con.SettingConf[op]; ok {
						if vs, vok := val.([]interface{}); vok {
							var valstrs []string
							for _, v := range vs {
								valstrs = append(valstrs, v.(string))
							}
							err := con.refreshElf(op, valstrs, binery)
							return con, err
						} else if vs, vok := val.(string); vok {
							err := con.refreshElf(op, []string{vs}, binery)
							return con, err
						} else {
							cerr := ErrNew(ErrMismatch, "elf operation is not right")
							return con, &cerr
						}
					}
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
		err := con.ContainerPaeudoShell()
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

func WalkContainerRoot(con *Container) ([]string, []string, *Error) {
	var libs []string
	var bineries []string
	files, err := walkfs(con.RootPath)
	if err != nil {
		return nil, nil, err
	}
	for _, file := range files {
		if strings.Contains(file, ".so") {
			libs = append(libs, file)
		} else {
			bineries = append(bineries, file)
		}
	}
	return bineries, libs, nil
}

func WalkSpecificDir(dir string) ([]string, *Error) {
	return walkfs(dir)
}

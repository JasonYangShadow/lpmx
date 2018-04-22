package container

import (
	"bufio"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/msgpack"
	. "github.com/jasonyangshadow/lpmx/paeudo"
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

type Container struct {
	Id                  string
	RootPath            string
	ConfigPath          string
	Status              int8
	LogPath             string
	ElfPatcherPath      string
	FakechrootPath      string
	SettingConf         map[string]interface{}
	StartTime           string
	ContainerName       string
	CreateUser          string
	MemcachedServerList string
	ShmFiles            string
	IpcFiles            string
	V                   *viper.Viper
}

func Run(dir string, config string) *Error {
	rootdir := fmt.Sprintf("%s/.lpmx", dir)
	var con Container
	con.RootPath = dir
	con.ConfigPath = rootdir

	defer func() {
		data, _ := StructMarshal(&con)
		WriteToFile(data, fmt.Sprintf("%s/.info", rootdir))
	}()

	if FolderExist(rootdir) {
		info := fmt.Sprintf("%s/.info", rootdir)
		if FileExist(info) {
			data, err := ReadFromFile(info)
			if err == nil {
				err := StructUnmarshal(data, &con)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.info doesn't exist", rootdir))
			return &cerr
		}
	} else {
		_, err := MakeDir(rootdir)
		if err != nil {
			return err
		}
		err = con.createContainer(config)
		if err != nil {
			return err
		}
		err = con.patchBineries()
		if err != nil {
			return err
		}
	}
	err := con.paeudoShell()
	if err != nil {
		return err
	}

	return nil
}

/**
container methods
**/

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

func (con *Container) paeudoShell() *Error {
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

func (con *Container) createContainer(config string) *Error {
	for strings.HasSuffix(con.ConfigPath, "/") {
		con.ConfigPath = strings.TrimSuffix(con.ConfigPath, "/")
	}
	err := con.createSysFolders(config)
	if err != nil {
		return err
	}
	return nil
}

func (con *Container) createSysFolders(config string) *Error {
	con.LogPath = fmt.Sprintf("%s/log", con.ConfigPath)
	con.ElfPatcherPath = fmt.Sprintf("%s/elf", con.ConfigPath)
	con.FakechrootPath = fmt.Sprintf("%s/fakechroot", con.ConfigPath)
	var err *Error
	con.CreateUser, err = Command("whoami")
	if err != nil {
		return err
	}
	con.V, con.SettingConf, err = LoadConfig(config)
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
	if FileExist("./patchelf") {
		_, err = CopyFile("./patchelf", fmt.Sprintf("%s/patchelf", con.ElfPatcherPath))
		if err != nil {
			return err
		}
	}
	if FileExist("./libfakechroot.so") {
		_, err = CopyFile("./libfakechroot.so", fmt.Sprintf("%s/libfakechroot.so", con.FakechrootPath))
		if err != nil {
			return err
		}
	}
	return nil
}

func (con *Container) patchBineries() *Error {
	bineries, _, err := walkContainerRoot(con)
	if err == nil {
		for _, op := range ELFOP {
			if data, ok := con.SettingConf[op]; ok {
				switch op {
				case ELFOP[0], ELFOP[1]:
					d1, o1 := data.([]interface{})
					if o1 == nil {
						for d11 := range d1 {
							d111, o111 := d11.(map[string]interface{})
							if o111 == nil {
								for k, v := range d111 {
									vs, _ := v.([]interface{})
									var libs []string
									for v1 := range vs {
										libs = append(libs, v1.(string))
									}
									err := con.refreshElf(op, libs, k)
									if err != nil {
										return err
									}
								}
							}
						}
					}
				case ELFOP[2], ELFOP[3]:
					d1, o1 := data.([]interface{})
					if o1 == nil {
						var rpaths []string
						for d11 := range d1 {
							rpaths = append(rpaths, d11.(string))
						}
						for binery := range bineries {
							err := con.refreshElf(op, rpaths, binery)
							if err != nil {
								return err
							}
						}
					}
				}
			}

			err := con.refreshElf(op, bineries, "")
		}
	}
	return err
}

/**
private functions
**/
func walkContainerRoot(con *Container) ([]string, []string, *Error) {
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

func walkSpecificDir(dir string) ([]string, *Error) {
	return walkfs(dir)
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

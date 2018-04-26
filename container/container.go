package container

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/memcache"
	. "github.com/jasonyangshadow/lpmx/msgpack"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RUNNING = iota
	STOPPED

	IDLENGTH = 10
)

var (
	ELFOP  = []string{"add_needed", "remove_needed", "add_rpath", "remove_rpath", "change_user"}
	STATUS = []string{"RUNNING", "STOPPED"}
)

type Sys struct {
	RootDir    string
	Containers map[string]interface{}
	LogPath    string
}

type Container struct {
	Id                  string
	RootPath            string
	ConfigPath          string
	Status              int
	LogPath             string
	ElfPatcherPath      string
	FakechrootPath      string
	SettingConf         map[string]interface{}
	SettingPath         string
	StartTime           string
	ContainerName       string
	CreateUser          string
	CurrentUser         string
	MemcachedServerList []string
	ShmFiles            string
	IpcFiles            string
	CurrentDir          string
	UserShell           string
	V                   *viper.Viper
}

func Init() *Error {
	currdir, _ := filepath.Abs(filepath.Dir("."))
	var sys Sys
	config := fmt.Sprintf("%s/.lpmxsys", currdir)
	sys.RootDir = config
	sys.LogPath = fmt.Sprintf("%s/log", sys.RootDir)

	defer func() {
		data, _ := StructMarshal(&sys)
		WriteToFile(data, fmt.Sprintf("%s/.info", sys.RootDir))
	}()

	if FolderExist(config) {
		err := readSys(sys.RootDir, &sys)
		if err != nil {
			return err
		}
	} else {
		_, err := MakeDir(sys.RootDir)
		if err != nil {
			return err
		}
		_, err = MakeDir(sys.LogPath)
		if err != nil {
			return err
		}
		sys.Containers = make(map[string]interface{})
	}
	return nil
}

func List() *Error {
	currdir, _ := filepath.Abs(filepath.Dir("."))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)
	if err == nil {
		fmt.Println(fmt.Sprintf("%s%40s%40s", "ContainerID", "RootPath", "Status"))
		for k, v := range sys.Containers {
			if cmap, ok := v.(map[string]interface{}); ok {
				fmt.Println(fmt.Sprintf("%s%40s%40s", k, cmap["RootPath"].(string), cmap["Status"].(string)))
			} else {
				cerr := ErrNew(ErrType, "sys.Containers type error")
				return &cerr
			}
		}
		return nil
	}
	return err
}

func Resume(id string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir("."))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)
	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				if val["Status"].(string) == STATUS[1] {
					err := Run(val["RootPath"].(string), val["SettingPath"].(string))
					if err != nil {
						return err
					}
				} else {
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running, can't resume", id))
					return &cerr
				}
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return &cerr
		}
		return nil
	}
	return err

}

func Destroy(id string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir("."))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

	defer func() {
		data, _ := StructMarshal(&sys)
		WriteToFile(data, fmt.Sprintf("%s/.info", sys.RootDir))
	}()

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				if val["Status"].(string) == STATUS[1] {
					cdir := fmt.Sprintf("%s/.lpmx", val["RootPath"])
					if FolderExist(cdir) {
						os.RemoveAll(cdir)
					}
					delete(sys.Containers, id)
				} else {
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running, can't destroy", id))
					return &cerr
				}
				return nil
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return &cerr
		}
		return nil
	}
	return err

}

func Run(dir string, config string) *Error {
	rootdir := fmt.Sprintf("%s/.lpmx", dir)
	var con Container
	con.RootPath = dir
	con.CurrentDir = dir
	con.ConfigPath = rootdir

	defer func() {
		data, _ := StructMarshal(&con)
		WriteToFile(data, fmt.Sprintf("%s/.info", rootdir))
		con.Status = STOPPED
		con.appendToSys()
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
		err = con.appendToSys()
		if err != nil {
			return err
		}
		err = con.setProgPrivileges()
		if err != nil {
			return err
		}
	}
	con.Status = RUNNING
	err := con.appendToSys()
	if err != nil {
		return err
	}
	err = con.bashShell()
	if err != nil {
		return err
	}

	return nil
}

func Set(id string, tp string, name string, value string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir("."))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				if val["Status"].(string) == STATUS[0] {
					err_new := ErrNew(ErrStatus, "container is running now, can't change the info, please stop it firstly")
					return &err_new
				}
				var con Container
				info := fmt.Sprintf("%s/.lpmx/.info", val["RootPath"].(string))
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
					cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.info doesn't exist", val["RootPath"].(string)))
					return &cerr
				}
				switch tp {
				case ELFOP[0], ELFOP[1], ELFOP[2], ELFOP[3]:
					values := strings.Split(value, ",")
					rerr := con.refreshElf(tp, values, name)
					if rerr != nil {
						return rerr
					}
				case ELFOP[4]:
					if strings.TrimSpace(name) != "user" {
						err_new := ErrNew(ErrType, "name should be 'user'")
						return &err_new
					}
					switch value {
					case "root":
						con.CurrentUser = "root"
					case "default":
						con.CurrentUser = con.CreateUser
					default:
						err_new := ErrNew(ErrType, "value should be either 'root' or 'default'")
						return &err_new
					}
				default:
					err_new := ErrNew(ErrType, "tp should be one of 'add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user'}")
					return &err_new
				}
				//write back
				data, _ := StructMarshal(&con)
				WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))

				return nil
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return &cerr
		}
		return nil
	}
	return err
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

func (con *Container) bashShell() *Error {
	if FolderExist(con.RootPath) {
		env := make(map[string]string)
		env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so", con.FakechrootPath)
		env["ContainerId"] = con.Id
		env["ContainerRoot"] = con.RootPath
		var err *Error
		if con.CurrentUser == "root" {
			err = ShellEnv("fakeroot", env, con.RootPath, con.UserShell)
		} else {
			err = ShellEnv(con.UserShell, env, con.RootPath)
		}
		if err != nil {
			return err
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
	con.Id = randomString(IDLENGTH)
	con.LogPath = fmt.Sprintf("%s/log", con.ConfigPath)
	con.ElfPatcherPath = fmt.Sprintf("%s/elf", con.ConfigPath)
	con.FakechrootPath = fmt.Sprintf("%s/fakechroot", con.ConfigPath)
	user, err := Command("whoami")
	if err != nil {
		return err
	}
	con.CreateUser = strings.TrimSuffix(user, "\n")
	con.CurrentUser = con.CreateUser
	con.V, con.SettingConf, err = LoadConfig(config)
	if err != nil {
		return err
	}
	con.SettingPath, _ = filepath.Abs(config)
	if sh, ok := con.SettingConf["user_shell"]; ok {
		strsh, _ := sh.(string)
		con.UserShell = strsh
	} else {
		con.UserShell = "/usr/bin/bash"
	}
	if mem, mok := con.SettingConf["memcache_list"]; mok {
		if mems, mems_ok := mem.([]interface{}); mems_ok {
			for _, memc := range mems {
				con.MemcachedServerList = append(con.MemcachedServerList, memc.(string))
			}
		}
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
					if d1, o1 := data.([]interface{}); o1 {
						for _, d1_1 := range d1 {
							if d1_11, o1_11 := d1_1.(interface{}); o1_11 {
								for k, v := range d1_11.(map[interface{}]interface{}) {
									if v1, vo1 := v.([]interface{}); vo1 {
										var libs []string
										for _, vv1 := range v1 {
											libs = append(libs, vv1.(string))
										}
										if k1, ok1 := k.(string); ok1 {
											if FileExist(k1) {
												err := con.refreshElf(op, libs, k1)
												if err != nil {
													return err
												}
											}
										}
									}
								}
							}
						}
					}
				case ELFOP[2], ELFOP[3]:
					if d1, o1 := data.([]interface{}); o1 {
						var rpaths []string
						for _, d1_1 := range d1 {
							rpaths = append(rpaths, d1_1.(string))
						}
						for _, binery := range bineries {
							if FileExist(binery) {
								err := con.refreshElf(op, rpaths, binery)
								if err != nil {
									return err
								}
							}
						}
					}
				}
			}
		}
	}
	return err
}

func (con *Container) appendToSys() *Error {
	currdir, _ := filepath.Abs(filepath.Dir("."))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

	if err == nil {
		if val, ok := sys.Containers[con.Id]; ok {
			if cmap, cok := val.(map[string]interface{}); cok {
				cmap["Status"] = STATUS[con.Status]
				cmap["RootPath"] = con.RootPath
				cmap["SettingPath"] = con.SettingPath
			} else {
				cerr := ErrNew(ErrType, "interface{} type assertation error")
				return &cerr
			}
		} else {
			cmap := make(map[string]string)
			cmap["Status"] = STATUS[con.Status]
			cmap["RootPath"] = con.RootPath
			cmap["SettingPath"] = con.SettingPath
			sys.Containers[con.Id] = cmap
		}
		data, _ := StructMarshal(&sys)
		err := WriteToFile(data, fmt.Sprintf("%s/.info", sys.RootDir))
		if err != nil {
			return err
		}
		return nil
	}
	return err

}

func (con *Container) setProgPrivileges() *Error {
	var mem *MemcacheInst
	var err *Error
	if len(con.MemcachedServerList) > 0 {
		mem, err = MInitServers(con.MemcachedServerList[0:]...)
	} else {
		mem, err = MInitServer()
	}
	if err == nil {
		if ac, ac_ok := con.SettingConf["allow_list"]; ac_ok {
			if aca, aca_ok := ac.([]interface{}); aca_ok {
				for _, ace := range aca {
					if acm, acm_ok := ace.(map[string]interface{}); acm_ok {
						for k, v := range acm {
							switch v.(type) {
							case string:
								mem.MSetStrValue(fmt.Sprintf("%s:%s:allow", con.Id, k), v.(string))
							case interface{}:
								value := ""
								for _, ve := range v.([]interface{}) {
									value = fmt.Sprintf("%s;%s", ve.(string), value)
								}
								mem.MSetStrValue(fmt.Sprintf("%s:%s:allow", con.Id, k), value)
							default:
								cerr := ErrNew(ErrType, "interface{} type assertation error")
								return &cerr
							}
						}
					}
				}
			}
		}
		return nil
	}
	fmt.Println(mem, err)
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
		if f.IsDir() && f.Name() == ".lpmx" {
			return filepath.SkipDir
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

func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func readSys(rootdir string, sys *Sys) *Error {
	info := fmt.Sprintf("%s/.info", rootdir)
	if FileExist(info) {
		data, err := ReadFromFile(info)
		if err != nil {
			return err
		}
		err = StructUnmarshal(data, sys)
		if err != nil {
			return err
		}
	} else {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.info doesn't exist, please use command 'lpmx init' firstly", rootdir))
		return &cerr
	}
	return nil
}

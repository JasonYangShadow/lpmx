package container

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/log"
	. "github.com/jasonyangshadow/lpmx/memcache"
	. "github.com/jasonyangshadow/lpmx/msgpack"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/rpc"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"github.com/spf13/viper"
	"net"
	"net/rpc"
	"os"
	"os/signal"
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
	ELFOP  = []string{"add_needed", "remove_needed", "add_rpath", "remove_rpath", "change_user", "add_privilege", "remove_privilege"}
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
	RPCPort             int
	RPCMap              map[int]string
	V                   *viper.Viper
}

type RPC struct {
	Env map[string]string
	Dir string
	Con *Container
}

func (server *RPC) RPCExec(req Request, res *Response) error {
	var pid int
	var err *Error
	if filepath.IsAbs(req.Cmd) {
		pid, err = ProcessContextEnv(req.Cmd, server.Env, server.Dir, req.Timeout, req.Args...)
	} else {
		req.Cmd = filepath.Join(server.Dir, "/", req.Cmd)
		pid, err = ProcessContextEnv(req.Cmd, server.Env, server.Dir, req.Timeout, req.Args...)
	}
	if err != nil {
		return err.Err
	}
	res.UId = RandomString(UIDLENGTH)
	res.Pid = pid
	server.Con.RPCMap[pid] = req.Cmd
	return nil
}

func (server *RPC) RPCQuery(req Request, res *Response) error {
	for k, _ := range server.Con.RPCMap {
		_, err := os.FindProcess(k)
		if err != nil {
			delete(server.Con.RPCMap, k)
		}
	}
	res.RPCMap = server.Con.RPCMap
	return nil
}

func (server *RPC) RPCDelete(req Request, res *Response) error {
	if _, ok := server.Con.RPCMap[req.Pid]; ok {
		process, err := os.FindProcess(req.Pid)
		if err == nil {
			perr := process.Signal(os.Interrupt)
			if perr == nil {
				delete(server.Con.RPCMap, req.Pid)
			} else {
				return perr
			}
		} else {
			delete(server.Con.RPCMap, req.Pid)
		}
	}
	return nil
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
		fmt.Println(fmt.Sprintf("%s%25s%25s%25s", "ContainerID", "RootPath", "Status", "RPC"))
		for k, v := range sys.Containers {
			if cmap, ok := v.(map[string]interface{}); ok {
				port := strings.TrimSpace(cmap["RPCPort"].(string))
				if port != "0" {
					conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), time.Millisecond*200)
					if err == nil && conn != nil {
						conn.Close()
						fmt.Println(fmt.Sprintf("%s%25s%25s%25s", k, cmap["RootPath"].(string), cmap["Status"].(string), cmap["RPCPort"].(string)))
					}
				} else {
					fmt.Println(fmt.Sprintf("%s%25s%25s%25s", k, cmap["RootPath"].(string), cmap["Status"].(string), "NA"))
				}
			} else {
				cerr := ErrNew(ErrType, "sys.Containers type error")
				return &cerr
			}
		}
		return nil
	}
	return err
}

func RPCExec(ip string, port string, timeout string, cmd string, args ...string) (*Response, *Error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		cerr := ErrNew(err, "tcp dial error")
		return nil, &cerr
	}
	var req Request
	var res Response
	req.Cmd = cmd
	req.Timeout = timeout
	var arg []string
	for _, a := range args {
		arg = append(arg, a)
	}
	req.Args = arg
	divCall := client.Go("RPC.RPCExec", req, &res, nil)
	go func() {
		<-divCall.Done
	}()
	return &res, nil
	/**err = client.Call("RPC.RPCExec", req, &res)**/
}

func RPCQuery(ip string, port string) (*Response, *Error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		cerr := ErrNew(err, "tcp dial error")
		return nil, &cerr
	}
	var req Request
	var res Response
	err = client.Call("RPC.RPCQuery", req, &res)
	if err != nil {
		cerr := ErrNew(err, "rpc call encounters error")
		return nil, &cerr
	}
	return &res, nil
}

func RPCDelete(ip string, port string, pid int) (*Response, *Error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		cerr := ErrNew(err, "tcp dial error")
		return nil, &cerr
	}
	var req Request
	var res Response
	req.Pid = pid
	err = client.Call("RPC.RPCDelete", req, &res)
	if err != nil {
		cerr := ErrNew(err, "rpc call encounters error")
		return nil, &cerr
	}
	return &res, nil
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
					err := Run(val["RootPath"].(string), val["SettingPath"].(string), false)
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

func Run(dir string, config string, passive bool) *Error {
	rootdir := fmt.Sprintf("%s/.lpmx", dir)
	var con Container
	con.RootPath = dir
	con.CurrentDir = dir
	con.ConfigPath = rootdir
	llog := new(Log)
	llog.LogInit(fmt.Sprintf("%s/log", con.RootPath))

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
					llog.LogFatal.Println("struct unmarshal error")
					return err
				}
			} else {
				llog.LogFatal.Println(fmt.Sprintf("read from file: %s error", info))
				return err
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.info doesn't exist", rootdir))
			return &cerr
		}
	} else {
		_, err := MakeDir(rootdir)
		if err != nil {
			llog.LogFatal.Println(fmt.Sprintf("mkdir error: %s", rootdir))
			return err
		}
		err = con.createContainer(config)
		if err != nil {
			llog.LogFatal.Println(fmt.Sprintf("create container error: %s", config))
			return err
		}
		err = con.patchBineries()
		if err != nil {
			llog.LogFatal.Println("patch bineries error")
			return err
		}
		err = con.appendToSys()
		if err != nil {
			llog.LogFatal.Println("append to sys info error")
			return err
		}
		err = con.setProgPrivileges()
		if err != nil {
			llog.LogFatal.Println("set program privilegies encoutners error")
			return err
		}
	}

	if passive {
		con.Status = RUNNING
		con.RPCPort = RandomPort(MIN, MAX)
		err := con.appendToSys()
		if err != nil {
			llog.LogFatal.Println("append to sys info error")
			return err
		}
		err = con.startRPCService(con.RPCPort)
		if err != nil {
			llog.LogFatal.Println("starting rpc service encounters error")
			return err
		}
	} else {
		con.Status = RUNNING
		err := con.appendToSys()
		if err != nil {
			llog.LogFatal.Println("append to sys info error")
			return err
		}
		err = con.bashShell()
		if err != nil {
			llog.LogFatal.Println("starting bash shell encounters error")
			return err
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		con.Status = STOPPED
		data, _ := StructMarshal(&con)
		WriteToFile(data, fmt.Sprintf("%s/.info", rootdir))
		con.Status = STOPPED
		con.appendToSys()
	}()
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
				tp = strings.ToLower(strings.TrimSpace(tp))
				switch tp {
				case ELFOP[5], ELFOP[6]:
					{
						err := setPrivilege(id, tp, name, value)
						if err != nil {
							return err
						}
					}
				case ELFOP[0], ELFOP[1], ELFOP[2], ELFOP[3], ELFOP[4]:
					{
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
						//read info ends,  get container from file

						switch tp {
						case ELFOP[0], ELFOP[1], ELFOP[2], ELFOP[3]:
							values := strings.Split(value, ",")
							rerr := con.refreshElf(tp, values, name)
							if rerr != nil {
								return rerr
							}
						case ELFOP[4]:
							if val["Status"].(string) == STATUS[0] {
								err_new := ErrNew(ErrStatus, "container is running now, can't change the user, please stop it firstly")
								return &err_new
							}
							if strings.TrimSpace(name) != "user" {
								err_new := ErrNew(ErrType, "name should be 'user'")
								return &err_new
							}
							switch value {
							case "root":
								con.CurrentUser = "root"
							case "chroot":
								con.CurrentUser = "chroot"
							case "default":
								con.CurrentUser = con.CreateUser
							default:
								err_new := ErrNew(ErrType, "value should be either 'root','default' or 'chroot'")
								return &err_new
							}
						default:
							err_new := ErrNew(ErrType, "tp should be one of 'add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user'}")
							return &err_new
						}

						//write back
						data, _ := StructMarshal(&con)
						WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))

					}

					return nil
				default:
					err_new := ErrNew(ErrType, "tp should be one of 'add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user', 'add_privilege','remove_privilege'}")
					return &err_new
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
		env["ContainerId"] = con.Id
		env["ContainerRoot"] = con.RootPath
		env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so", con.FakechrootPath)
		var err *Error
		if con.CurrentUser == "root" {
			err = ShellEnv("fakeroot", env, con.RootPath, con.UserShell)
		} else if con.CurrentUser == "chroot" {
			env["ContainerMode"] = "chroot"
			shell := fmt.Sprintf("%s/%s", con.RootPath, con.UserShell)
			if !FileExist(shell) {
				_, err = MakeDir(filepath.Dir(shell))
				if err != nil {
					return err
				}
				_, err = CopyFile(con.UserShell, shell)
				if err != nil {
					return err
				}
			}
			err = ShellEnv("fakeroot", env, con.RootPath, "chroot", con.RootPath, con.UserShell)
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
	con.Id = RandomString(IDLENGTH)
	con.LogPath = fmt.Sprintf("%s/log", con.ConfigPath)
	con.ElfPatcherPath = fmt.Sprintf("%s/elf", con.ConfigPath)
	con.FakechrootPath = fmt.Sprintf("%s/fakechroot", con.ConfigPath)
	user, err := Command("whoami")
	if err != nil {
		return err
	}
	con.CreateUser = strings.TrimSuffix(user, "\n")
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
	if c_user, c_ok := con.SettingConf["default_user"]; c_ok {
		con.CurrentUser = c_user.(string)
	} else {
		con.CurrentUser = con.CreateUser
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
										vv1_abs, _ := filepath.Abs(vv1.(string))
										libs = append(libs, vv1_abs)
									}
									if k1, ok1 := k.(string); ok1 {
										k1_abs, _ := filepath.Abs(k1)
										if FileExist(k1_abs) {
											err := con.refreshElf(op, libs, k1_abs)
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
					for _, d1_1 := range d1 {
						if d1_11, o1_11 := d1_1.(interface{}); o1_11 {
							for k, v := range d1_11.(map[interface{}]interface{}) {

								var libs []string
								if v1, vo1 := v.([]interface{}); vo1 {
									for _, vv1 := range v1 {
										vv1_abs, _ := filepath.Abs(vv1.(string))
										libs = append(libs, vv1_abs)
									}
								}

								k_abs, _ := filepath.Abs(k.(string))
								if FileExist(k_abs) {
									err := con.refreshElf(op, libs, k_abs)
									if err != nil {
										return err
									}
								}
								if FolderExist(k_abs) {
									bineries, err := walkSpecificDir(k_abs)
									if err != nil {
										return err
									}
									for _, binery := range bineries {
										if FileExist(binery) {
											var paths []string
											paths = append(paths, v.(string))
											err := con.refreshElf(op, paths, binery)
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
			}
		}
	}
	return nil
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
				cmap["RPCPort"] = fmt.Sprintf("%d", con.RPCPort)
			} else {
				cerr := ErrNew(ErrType, "interface{} type assertation error")
				return &cerr
			}
		} else {
			cmap := make(map[string]string)
			cmap["Status"] = STATUS[con.Status]
			cmap["RootPath"] = con.RootPath
			cmap["SettingPath"] = con.SettingPath
			cmap["RPCPort"] = fmt.Sprintf("%d", con.RPCPort)
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
		if err != nil {
			return err
		}
	} else {
		mem, err = MInitServer()
		if err != nil {
			return err
		}
	}
	if err == nil {
		if ac, ac_ok := con.SettingConf["allow_list"]; ac_ok {
			if aca, aca_ok := ac.([]interface{}); aca_ok {
				for _, ace := range aca {
					if acm, acm_ok := ace.(map[interface{}]interface{}); acm_ok {
						for k, v := range acm {
							switch v.(type) {
							case string:
								mem_err := mem.MSetStrValue(fmt.Sprintf("%s:%s", con.Id, k), v.(string))
								if mem_err != nil {
									return mem_err
								}
							case interface{}:
								if acs, acs_ok := v.([]interface{}); acs_ok {
									value := ""
									for _, acl := range acs {
										value = fmt.Sprintf("%s;%s", acl.(string), value)
									}
									mem_err := mem.MSetStrValue(fmt.Sprintf("%s:%s", con.Id, k), value)
									if mem_err != nil {
										return mem_err
									}
								}
							default:
								acm_err := ErrNew(ErrType, fmt.Sprintf("type is not right, assume: map[interfacer{}]interface{}, real: %v", ace))
								return &acm_err
							}
						}
					}
				}
			} else {
				aca_err := ErrNew(ErrType, fmt.Sprintf("type is not right, assume: []interface{}, real: %v", ac))
				return &aca_err
			}
		}
		return nil
	} else {
		mem_err := ErrNew(err, "memcache server init error")
		return &mem_err
	}
	return err
}

func (con *Container) startRPCService(port int) *Error {
	con.RPCMap = make(map[int]string)
	conn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		cerr := ErrNew(err, "start rpc service encounters error")
		return &cerr
	}
	env := make(map[string]string)
	env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so", con.FakechrootPath)
	env["ContainerId"] = con.Id
	env["ContainerRoot"] = con.RootPath
	r := new(RPC)
	r.Env = env
	r.Dir = con.RootPath
	r.Con = con
	rpc.Register(r)
	rpc.Accept(conn)
	return nil
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

func setPrivilege(id string, tp string, name string, value string) *Error {
	mem, err := MInitServer()
	if err != nil {
		return err
	}

	if tp == ELFOP[5] {
		err := mem.MSetStrValue(fmt.Sprintf("%s:%s", id, name), value)
		if err != nil {
			return err
		}
	} else {
		err := mem.MDeleteByKey(fmt.Sprintf("%s:%s", id, name))
		if err != nil {
			return err
		}
	}
	return nil
}

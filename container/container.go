package container

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/JasonYangShadow/lpmx/docker"
	. "github.com/JasonYangShadow/lpmx/elf"
	. "github.com/JasonYangShadow/lpmx/error"
	. "github.com/JasonYangShadow/lpmx/filecache"
	. "github.com/JasonYangShadow/lpmx/log"
	. "github.com/JasonYangShadow/lpmx/memcache"
	. "github.com/JasonYangShadow/lpmx/msgpack"
	. "github.com/JasonYangShadow/lpmx/paeudo"
	. "github.com/JasonYangShadow/lpmx/pid"
	. "github.com/JasonYangShadow/lpmx/rpc"
	. "github.com/JasonYangShadow/lpmx/singularity"
	. "github.com/JasonYangShadow/lpmx/utils"
	. "github.com/JasonYangShadow/lpmx/yaml"
	"github.com/sirupsen/logrus"
)

const (
	IDLENGTH = 10
)

var (
	ELFOP                   = []string{"add_allow_priv", "remove_allow_priv", "add_deny_priv", "remove_deny_priv", "add_map", "remove_map", "add_exec", "remove_exec"}
	LD                      = []string{"/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2", "/lib/ld.so", "/lib64/ld-linux-x86-64.so.2", "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.1", "/lib64/ld-linux-x86-64.so.1", "/lib/ld-linux.so.2", "/lib/ld-linux.so.1"}
	LD_LIBRARY_PATH_DEFAULT = []string{"lib", "lib64", "lib/x86_64-linux-gnu", "usr/lib/x86_64-linux-gnu", "usr/lib", "usr/local/lib", "usr/lib64", "usr/local/lib64"}
	CACHE_FOLDER            = []string{"/var/cache/apt/archives"}
	UNSTALL_FOLDER          = []string{".lpmxsys", "sync", "bin", ".lpmxdata", "package"}
	ENGINE_TYPE             = []string{"SGE"}
)

//located inside $/.lpmxsys/.info
type Sys struct {
	RootDir      string // the abs path of folder .lpmxsys
	Containers   map[string]interface{}
	LogPath      string
	MemcachedPid string
}

//located inside $/.lpmxdata/image/tag/workspace/.lpmx/.info
type Container struct {
	Id                  string
	RootPath            string
	ConfigPath          string
	ImageBase           string
	BaseType            string
	Layers              string //e.g, rw:layer1:layer2
	BaseLayerPath       string
	LogPath             string
	ElfPatcherPath      string
	PatchedELFLoader    string
	SettingConf         map[string]interface{}
	SettingPath         string
	SysDir              string //dir of lpmx set by appendToSys function, the directory containing dependencies
	StartTime           string
	ContainerName       string
	CreateUser          string
	CurrentUser         string
	MemcachedServerList []string
	ExposeExe           string
	UserShell           string
	RPCPort             int
	RPCMap              map[int]string
	PidFile             string
	Pid                 int
	DataSyncFolder      string //sync folder with host
	DataSyncMap         string //sync folder mapping info(host:contain)
	Engine              string //engine type used on the host
	Execmaps            string //executables mapping info
}

type RPC struct {
	Env map[string]string
	Dir string
	Con *Container
}

//used for storing all images, located inside $/.lpmxdata/.info
type Image struct {
	RootDir string
	Images  map[string]interface{}
}

//used for offline image installation, located inside $/.lpmxdata/image/tag/.info
type ImageInfo struct {
	Name      string
	ImageType string
	LayersMap map[string]int64 //map containing layers and their sizes
	Layers    string           //should be original order, used for extraction
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

func Init(reset bool, deppath string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	config := fmt.Sprintf("%s/.lpmxsys", currdir)

	//delete everything
	if reset {
		_, cerr := RemoveAll(config)
		if cerr != nil {
			return cerr
		}
	}

	sys.RootDir = config
	sys.LogPath = fmt.Sprintf("%s/log", sys.RootDir)

	defer func() {
		data, _ := StructMarshal(&sys)
		WriteToFile(data, fmt.Sprintf("%s/.info", sys.RootDir))
	}()

	configfile := fmt.Sprintf("%s/.info", sys.RootDir)
	if FileExist(configfile) {
		err := unmarshalObj(sys.RootDir, &sys)
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
		_, err = MakeDir(fmt.Sprintf("%s/bin", currdir))
		if err != nil {
			return err
		}
		sys.Containers = make(map[string]interface{})

		//download memcached related files based on host os info
		dist, release, cerr := GetHostOSInfo()
		if cerr != nil {
			dist = "default"
			release = "default"
		}

		if dist == "" {
			dist = "default"
		}

		if release == "" {
			release = "default"
		}

		if deppath == "" {
			deppath = fmt.Sprintf("%s/dependency.tar.gz", sys.RootDir)
		}
		if !FileExist(deppath) {
			yaml := fmt.Sprintf("%s/distro.management.yml", sys.RootDir)
			err = DownloadFilefromGithubPlus(dist, release, "dependency.tar.gz", SETTING_URL, sys.RootDir, yaml)
			if err != nil {
				return err
			}
		}

		fmt.Println("Uncompressing downloaded dependency.tar.gz")
		err = Untar(deppath, sys.RootDir)
		if err != nil {
			return err
		}

		fmt.Println("Permission checking")
	}

	path := os.Getenv("PATH")

	if !strings.HasSuffix(path, currdir) {
		path = fmt.Sprintf("%s:%s", path, currdir)
	}

	exposed_bin := fmt.Sprintf("%s/bin", currdir)
	if !strings.Contains(path, exposed_bin) {
		path = fmt.Sprintf("%s:%s", exposed_bin, path)
	}

	path_var := fmt.Sprintf("PATH=%s", path)
	bashrc := fmt.Sprintf("%s/.bashrc", os.Getenv("HOME"))
	ferr := AddVartoFile(path_var, bashrc)
	if ferr != nil {
		return ferr
	}
	os.Setenv("PATH", path)

	host_ld_env := os.Getenv("LD_LIBRARY_PATH")
	if !strings.HasPrefix(host_ld_env, sys.RootDir) {
		if host_ld_env != "" {
			host_ld_env = fmt.Sprintf("%s:%s", sys.RootDir, host_ld_env)
		} else {
			host_ld_env = sys.RootDir
		}
		ld_var := fmt.Sprintf("LD_LIBRARY_PATH=%s", host_ld_env)
		ferr = AddVartoFile(ld_var, bashrc)
		if ferr != nil {
			return ferr
		}
		os.Setenv("LD_LIBRARY_PATH", host_ld_env)
	}

	if ok, _, _ := GetProcessIdByName("memcached"); !ok {
		fmt.Println("starting memcached process")
		cerr := CheckAndStartMemcache()
		if cerr != nil {
			return cerr
		}
	}
	sys.MemcachedPid = fmt.Sprintf("%s/.memcached.pid", sys.RootDir)

	return nil
}

func Uninstall() *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	if ok, pid, _ := GetProcessIdByName("memcached"); ok {
		fmt.Println("stopping memcached instance...")
		err := KillProcessByPid(pid)
		if err != nil {
			return err
		}
	}

	for _, i := range UNSTALL_FOLDER {
		_, err := RemoveAll(fmt.Sprintf("%s/%s", currdir, i))
		if err != nil {
			return err
		}
	}
	return nil
}

func Update() *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	config := fmt.Sprintf("%s/.lpmxsys", currdir)
	if !FolderExist(config) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("could not find config folder in %s, something goes wrong, please uninstall and reset", currdir))
		return cerr
	}

	//check if there are containers running
	var sys Sys
	err = unmarshalObj(config, &sys)
	if err == nil {
		//range containers
		for _, value := range sys.Containers {
			if vval, vok := value.(map[string]interface{}); vok {
				config_path := vval["ConfigPath"].(string)

				var con Container
				err = unmarshalObj(config_path, &con)
				if err != nil {
					return err
				}

				pidfile := fmt.Sprintf("%s/container.pid", path.Dir(con.RootPath))

				if pok, _ := PidIsActive(pidfile); pok {
					pid, _ := PidValue(pidfile)
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner is running with pid: %d, can't update(please stop all running containers in order to update libraries)", pid))
					return cerr
				}
			} else {
				cerr := ErrNew(ErrType, "container type is not map[string]interface{}")
				return cerr
			}
		}
	} else {
		return err
	}

	dist, release, cerr := GetHostOSInfo()
	if cerr != nil {
		dist = "default"
		release = "default"
	}

	if dist == "" {
		dist = "default"
	}

	if release == "" {
		release = "default"
	}

	deppath := fmt.Sprintf("%s/dependency.tar.gz", config)
	if FileExist(deppath) {
		_, ferr := RemoveFile(deppath)
		if ferr != nil {
			return ferr
		}
	}
	if !FileExist(deppath) {
		yaml := fmt.Sprintf("%s/distro.management.yml", sys.RootDir)
		err = DownloadFilefromGithubPlus(dist, release, "dependency.tar.gz", SETTING_URL, config, yaml)
		if err != nil {
			return err
		}
		//create temp folder
		temp, terr := CreateTempDir(config)
		if terr != nil {
			return terr
		}

		defer func() {
			if FolderExist(temp) {
				os.RemoveAll(temp)
			}
		}()

		err = Untar(deppath, temp)
		if err != nil {
			return err
		}

		err = Rename(fmt.Sprintf("%s/libfakechroot.so", temp), fmt.Sprintf("%s/libfakechroot.so", config))
		if err != nil {
			return err
		}

		//Done
		return nil
	}

	cerr = ErrNew(ErrExist, fmt.Sprintf("could not find %s, terminated", deppath))
	return cerr
}

func List(ListName string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)
	if err == nil {
		fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", "ContainerID", "ContainerName", "Status", "PID", "RPC", "BaseType", "Image"))
		for k, v := range sys.Containers {
			if cmap, ok := v.(map[string]interface{}); ok {
				//filter with name
				if cmap["ContainerName"].(string) != "" && ListName != "" && cmap["ContainerName"].(string) != ListName {
					continue
				}
				//get each container location
				root := path.Dir(cmap["RootPath"].(string))

				pid := -1
				//check if container is running
				if pok, _ := PidIsActive(fmt.Sprintf("%s/container.pid", root)); pok {
					pid, _ = PidValue(fmt.Sprintf("%s/container.pid", root))
				}

				//RPC MODE
				if cmap["RPC"] != nil && cmap["RPC"].(string) != "0" {
					conn, err := net.DialTimeout("tcp", net.JoinHostPort("", cmap["RPC"].(string)), time.Millisecond*200)
					if err == nil && conn != nil {
						conn.Close()
						if pid != -1 {
							fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "RUNNING", strconv.Itoa(pid), cmap["RPC"].(string), cmap["BaseType"].(string), cmap["Image"].(string)))
						} else {
							fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "STOPPED", "NA", cmap["RPC"].(string), cmap["BaseType"].(string), cmap["Image"].(string)))
						}
					}
				} else {
					if pid != -1 {
						fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "RUNNING", strconv.Itoa(pid), "NA", cmap["BaseType"].(string), cmap["Image"].(string)))
					} else {
						fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "STOPPED", "NA", "NA", cmap["BaseType"].(string), cmap["Image"].(string)))
					}
				}
			} else {
				cerr := ErrNew(ErrType, "sys.Containers type error")
				return cerr
			}
		}
		return nil
	}

	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err
}

func RPCExec(ip string, port string, timeout string, cmd string, args ...string) (*Response, *Error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		cerr := ErrNew(err, "tcp dial error")
		return nil, cerr
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
		return nil, cerr
	}
	var req Request
	var res Response
	err = client.Call("RPC.RPCQuery", req, &res)
	if err != nil {
		cerr := ErrNew(err, "rpc call encounters error")
		return nil, cerr
	}
	return &res, nil
}

func RPCDelete(ip string, port string, pid int) (*Response, *Error) {
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		cerr := ErrNew(err, "tcp dial error")
		return nil, cerr
	}
	var req Request
	var res Response
	req.Pid = pid
	err = client.Call("RPC.RPCDelete", req, &res)
	if err != nil {
		cerr := ErrNew(err, "rpc call encounters error")
		return nil, cerr
	}
	return &res, nil
}

func Resume(id string, engine bool, args ...string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)
	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				config_path := val["ConfigPath"].(string)

				var con Container
				err = unmarshalObj(config_path, &con)
				if err != nil {
					return err
				}
				pidfile := fmt.Sprintf("%s/container.pid", path.Dir(con.RootPath))

				if pok, _ := PidIsActive(pidfile); !pok {
					configmap := make(map[string]interface{})
					configmap["dir"] = con.RootPath
					configmap["config"] = con.SettingPath
					configmap["passive"] = false
					configmap["docker"] = true
					configmap["layers"] = con.Layers
					configmap["id"] = con.Id
					configmap["image"] = con.ImageBase
					configmap["baselayerpath"] = con.BaseLayerPath
					configmap["elf_loader"] = con.PatchedELFLoader
					configmap["parent_dir"] = filepath.Dir(con.RootPath)
					configmap["sync_folder"] = con.DataSyncFolder
					configmap["sync_ori_folder"] = con.DataSyncMap
					configmap["imagetype"] = con.BaseType
					configmap["engine"] = con.Engine

					//only if the user explicitly set enable_engine, then we skip enabling it
					if engine {
						configmap["enable_engine"] = "true"
					}
					err := Run(&configmap, args...)
					if err != nil {
						return err
					}
				} else {
					pid, _ := PidValue(pidfile)
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running with pid: %d, can't resume", id, pid))
					return cerr
				}
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}

	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err

}

func Destroy(id string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)

	defer func() {
		data, _ := StructMarshal(&sys)
		WriteToFile(data, fmt.Sprintf("%s/.info", sys.RootDir))
	}()

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				root := path.Dir(val["RootPath"].(string))

				pid := -1
				//check if container is running
				if pok, _ := PidIsActive(fmt.Sprintf("%s/container.pid", root)); pok {
					pid, _ = PidValue(fmt.Sprintf("%s/container.pid", root))
				}

				if pid == -1 {
					rootdir, _ := val["RootPath"].(string)
					rootdir = path.Dir(rootdir)
					RemoveAll(rootdir)
					//here we delete default sync folder
					data_sync_folder := fmt.Sprintf("%s/sync/%s", currdir, id)
					RemoveAll(data_sync_folder)
					delete(sys.Containers, id)
				} else {
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running with pid: %d, can't destroy", id, pid))
					return cerr
				}
				return nil
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}

	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err

}

func Run(configmap *map[string]interface{}, args ...string) *Error {
	//this map is used for dynamically controlling the generation of env vars
	envmap := make(map[string]string)

	//dir is rw folder of container
	dir, _ := (*configmap)["dir"].(string)
	config, _ := (*configmap)["config"].(string)
	passive, _ := (*configmap)["passive"].(bool)

	//parent dir is the folder containing rw, base layers and ld.so.patch
	parent_dir, _ := (*configmap)["parent_dir"].(string)
	rootdir := fmt.Sprintf("%s/.lpmx", parent_dir)

	var con Container
	con.Id = (*configmap)["id"].(string)
	con.BaseType = (*configmap)["imagetype"].(string)
	con.ImageBase = (*configmap)["image"].(string)
	con.Layers = (*configmap)["layers"].(string)
	con.BaseLayerPath = (*configmap)["baselayerpath"].(string)
	con.PatchedELFLoader = (*configmap)["elf_loader"].(string)
	con.DataSyncFolder = (*configmap)["sync_folder"].(string)
	con.DataSyncMap = (*configmap)["sync_ori_folder"].(string)
	con.RootPath = dir
	con.ConfigPath = rootdir
	con.SettingPath = config
	con.Engine = (*configmap)["engine"].(string)
	if (*configmap)["container_name"] == nil {
		(*configmap)["container_name"] = ""
	}
	con.ContainerName = (*configmap)["container_name"].(string)

	//save execmaps
	if _, eok := (*configmap)["execmaps"]; eok {
		con.Execmaps = (*configmap)["execmaps"].(string)
		envmap["execmaps"] = (*configmap)["execmaps"].(string)
	}

	//enable batch engine if needed
	if _, eok := (*configmap)["enable_engine"]; eok {
		envmap["engine"] = "TRUE"
	}

	defer func() {
		con.Pid = -1
		data, _ := StructMarshal(&con)
		WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))
		con.appendToSys()
	}()

	if FolderExist(con.ConfigPath) {
		info := fmt.Sprintf("%s/.info", con.ConfigPath)
		if FileExist(info) {
			data, err := ReadFromFile(info)
			if err == nil {
				err := StructUnmarshal(data, &con)
				if err != nil {
					err.AddMsg("struct unmarshal error")
					return err
				}
			} else {
				err.AddMsg(fmt.Sprintf("can't read configuration file from %s", info))
				return err
			}
		} else {
			RemoveAll(con.ConfigPath)
			err := con.setupContainer()
			if err != nil {
				return err
			}
		}
	} else {
		err := con.setupContainer()
		if err != nil {
			return err
		}
	}

	//here we need to inject executable mapping info
	if len(con.Execmaps) > 0 {
		for _, item := range strings.Split(con.Execmaps, ":") {
			k := strings.Split(item, "=")[0]
			v := strings.Split(item, "=")[1]
			setExec(con.Id, ELFOP[6], v, k, "", false)
		}
	}

	if passive {
		con.RPCPort = RandomPort(MIN, MAX)
		err := con.appendToSys()
		if err != nil {
			return err
		}
		err = con.startRPCService(con.RPCPort)
		if err != nil {
			err.AddMsg("starting rpc service encounters error")
			return err
		}
	} else {
		err := con.appendToSys()
		if err != nil {
			err.AddMsg("append to sys info error")
			return err
		}
		err = con.bashShell(envmap, args...)
		if err != nil {
			err.AddMsg("starting bash shell encounters error")
			return err
		}
	}

	return nil
}

func Get(id string, name string, mode bool) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)

	if err == nil {
		if _, ok := sys.Containers[id]; ok {
			fmt.Println(fmt.Sprintf("|%-s|%-30s|%-20s|%-20s|%-10s|%-20s|", "ContainerID", "PROGRAM", "ALLOW_PRIVILEGES", "DENY_PRIVILEGES", "REMAP", "ExecMap"))
			a_val, _ := getPrivilege(id, name, sys.MemcachedPid, true)
			d_val, _ := getPrivilege(id, name, sys.MemcachedPid, false)
			m_val, _ := getMap(id, name, sys.MemcachedPid, mode)
			e_val, _ := getExec(id, name, sys.MemcachedPid, mode)
			fmt.Println(fmt.Sprintf("|%-s|%-30s|%-20s|%-20s|%-10s|%-20s|", id, name, a_val, d_val, m_val, e_val))
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
	}

	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err
}

func Set(id string, tp string, name string, value string, mode bool) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if _, vok := v.(map[string]interface{}); vok {
				tp = strings.ToLower(strings.TrimSpace(tp))
				switch tp {
				case ELFOP[6], ELFOP[7]:
					{
						err := setExec(id, tp, name, value, sys.MemcachedPid, mode)
						if err != nil {
							return err
						}
					}
				case ELFOP[4], ELFOP[5]:
					{
						err := setMap(id, tp, name, value, sys.MemcachedPid, mode)
						if err != nil {
							return err
						}
					}
				case ELFOP[0], ELFOP[1]:
					{
						err := setPrivilege(id, tp, name, value, sys.MemcachedPid, true)
						if err != nil {
							return err
						}
					}
				case ELFOP[2], ELFOP[3]:
					{
						err := setPrivilege(id, tp, name, value, sys.MemcachedPid, false)
						if err != nil {
							return err
						}
					}
					return nil
				default:
					err_new := ErrNew(ErrType, "tp should be one of 'add_exec','remove_exec','add_map','remove_map'")
					return err_new
				}

			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}

	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err
}

func DockerSearch(name string) ([]string, *Error) {
	tags, err := ListTags("", "", name)
	return tags, err
}

func DockerPackage(name string, user string, pass string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	tempdir := fmt.Sprintf("%s/.temp", currdir)
	//we delete temp dir if it exists at the end of the function
	defer func() {
		if FolderExist(tempdir) {
			os.RemoveAll(tempdir)
		}
	}()

	packagedir := fmt.Sprintf("%s/package", currdir)
	if !FolderExist(packagedir) {
		MakeDir(packagedir)
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err != nil {
		return err
	}

	//create temp dir for temp docinfo file
	tdir, terr := CreateTempDir(tempdir)
	if terr != nil {
		return terr
	}

	if dvalue, dok := doc.Images[name]; dok {
		if dmap, dmok := dvalue.(map[string]interface{}); dmok {
			var filelist []string
			filelist = append(filelist, dmap["config"].(string))
			//here we check if orig_layer_order exists
			var layers []string
			if val, vok := dmap["orig_layer_order"]; vok {
				layers = strings.Split(val.(string), ":")
			} else {
				layers = strings.Split(dmap["layer_order"].(string), ":")
			}
			layer_base := dmap["image"].(string)

			var docinfo ImageInfo
			docinfo.Name = name
			docinfo.ImageType = "Docker"
			docinfo.LayersMap = make(map[string]int64)
			var docinfo_layers []string

			for _, layer := range layers {
				sha256 := path.Base(layer)
				//layer here is the format of sha256.tar.gz
				docinfo_layers = append(docinfo_layers, sha256)
				file_name := fmt.Sprintf("%s/%s", layer_base, sha256)
				filelist = append(filelist, file_name)
				size, serr := GetFileLength(file_name)
				if serr != nil {
					return serr
				}
				docinfo.LayersMap[sha256] = size
			}
			docinfo.Layers = strings.Join(docinfo_layers, ":")

			dinfodata, _ := StructMarshal(docinfo)
			err = WriteToFile(dinfodata, fmt.Sprintf("%s/.info", tdir))
			if err != nil {
				return err
			}

			filelist = append(filelist, fmt.Sprintf("%s/.info", tdir))
			LOGGER.WithFields(logrus.Fields{
				"docinfo":  docinfo,
				"filelist": filelist,
			}).Debug("DockerPackage docinfo and filelist to tar")

			cerr := TarFiles(filelist, packagedir, name)
			if cerr != nil {
				return cerr
			}
		} else {
			cerr := ErrNew(ErrMismatch, "type mismatched")
			return cerr
		}
	} else {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("no such image: %s", name))
		return cerr
	}
	return nil
}

func DockerAdd(file string) *Error {
	if !FileExist(file) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does not exist", file))
		return cerr
	}
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	tempdir := fmt.Sprintf("%s/.temp", currdir)
	//we delete temp dir if it exists at the end of the function
	defer func() {
		if FolderExist(tempdir) {
			os.RemoveAll(tempdir)
		}
	}()
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err != nil && err.Err != ErrNExist {
		return err
	}
	if err != nil && err.Err == ErrNExist {
		ret, err := MakeDir(rootdir)
		doc.RootDir = rootdir
		doc.Images = make(map[string]interface{})
		if !ret {
			return err
		}
	}

	//start untaring offline file
	//create tmp folder
	dir, derr := CreateTempDir(tempdir)
	if derr != nil {
		return derr
	}

	//Uncompressing
	cerr := Untar(file, dir)
	if cerr != nil {
		return cerr
	}
	//read .info
	if !FileExist(fmt.Sprintf("%s/.info", dir)) {
		cerr := ErrNew(ErrNExist, ".info metafile does not exist inside package, maybe the package is not created by 'lpmx docker package' command")
		return cerr
	}
	var docinfo ImageInfo
	cerr = unmarshalObj(dir, &docinfo)
	if cerr != nil {
		return cerr
	}
	if _, ok := doc.Images[docinfo.Name]; ok {
		return nil
	} else {
		tdata := strings.Split(docinfo.Name, ":")
		tname := tdata[0]
		ttag := tdata[1]
		mdata := make(map[string]interface{})
		mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", doc.RootDir, tname, ttag)
		mdata["config"] = fmt.Sprintf("%s/setting.yml", mdata["rootdir"])
		mdata["image"] = fmt.Sprintf("%s/.image", rootdir)
		mdata["imagetype"] = "Docker"
		image_dir, _ := mdata["image"].(string)

		if !FolderExist(mdata["rootdir"].(string)) {
			MakeDir(mdata["rootdir"].(string))
		}

		if !FolderExist(mdata["image"].(string)) {
			MakeDir(mdata["image"].(string))
		}

		//move layers
		layers := strings.Split(docinfo.Layers, ":")
		for _, lay := range layers {
			lay_path := fmt.Sprintf("%s/%s", dir, lay)
			if !FileExist(lay_path) {
				cerr := ErrNew(ErrNExist, fmt.Sprintf("%s layer does not exist", lay_path))
				return cerr
			}
			lay_new_path := fmt.Sprintf("%s/%s", mdata["image"], lay)
			if !FileExist(lay_new_path) {
				err := os.Rename(lay_path, lay_new_path)
				if err != nil {
					cerr := ErrNew(err, fmt.Sprintf("could not move file %s to %s", lay_path, lay_new_path))
					return cerr
				}
			}
		}

		//move setting.yml
		config_path := fmt.Sprintf("%s/setting.yml", dir)
		if !FileExist(config_path) {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does not exist", config_path))
			return cerr
		}
		err := os.Rename(config_path, mdata["config"].(string))
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not move file %s to %s", config_path, mdata["config"]))
			return cerr
		}

		//here we have to restore absolute path
		layersmap := make(map[string]int64)
		for k, v := range docinfo.LayersMap {
			layersmap[fmt.Sprintf("%s/%s", mdata["image"].(string), k)] = v
		}
		mdata["layer"] = layersmap
		mdata["layer_order"] = docinfo.Layers

		workspace := fmt.Sprintf("%s/workspace", mdata["rootdir"])
		if !FolderExist(workspace) {
			MakeDir(workspace)
		}
		mdata["workspace"] = workspace

		//extract layers
		base := fmt.Sprintf("%s/.base", rootdir)
		if !FolderExist(base) {
			MakeDir(base)
		}
		mdata["base"] = base

		layer_order := strings.Split(docinfo.Layers, ":")
		for _, k := range layer_order {
			tar_path := fmt.Sprintf("%s/%s", image_dir, k)
			layerfolder := fmt.Sprintf("%s/%s", mdata["base"], k)
			if !FolderExist(layerfolder) {
				MakeDir(layerfolder)
				err := Untar(tar_path, layerfolder)
				if err != nil {
					return err
				}
			}
		}

		//move .info
		info_path := fmt.Sprintf("%s/.info", dir)
		info_new_path := fmt.Sprintf("%s/.info", mdata["rootdir"])
		err = os.Rename(info_path, info_new_path)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not move file %s to %s", info_path, info_new_path))
			return cerr
		}

		doc.Images[docinfo.Name] = mdata

		LOGGER.WithFields(logrus.Fields{
			"doc": doc,
		}).Debug("DockerAdd update image info")
		ddata, _ := StructMarshal(doc)
		cerr := WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
		if cerr != nil {
			return cerr
		}
		return nil
	}
}

func DockerCommit(id, newname, newtag string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	//first check whether the container is running
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)
	tempdir := fmt.Sprintf("%s/.temp", currdir)

	//here we delete tempdir if it exists at the end of this function
	defer func() {
		if FolderExist(tempdir) {
			os.RemoveAll(tempdir)
		}
	}()
	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				config_path := val["ConfigPath"].(string)

				var con Container
				err = unmarshalObj(config_path, &con)
				if err != nil {
					return err
				}
				pidfile := fmt.Sprintf("%s/container.pid", path.Dir(con.RootPath))

				if pok, _ := PidIsActive(pidfile); !pok {
					//parameters are src/target folder, target file name, and layer paths
					//check new image already exists?
					rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
					var doc Image
					err := unmarshalObj(rootdir, &doc)
					if err != nil {
						return err
					}
					if _, ok := doc.Images[fmt.Sprintf("%s:%s", newname, newtag)]; ok {
						cerr := ErrNew(ErrExist, fmt.Sprintf("%s:%s already exists, please choose another name and tag", newname, newtag))
						return cerr
					}

					//step0: before taring rw layer, remove unecessary folders
					for _, cache := range CACHE_FOLDER {
						cache = fmt.Sprintf("%s%s", con.RootPath, cache)
						if FolderExist(cache) {
							RemoveAll(cache)
						}
					}
					//moving /etc/group /etc/passwd /tmp folder to temp folder
					cache_temp_dir, cache_err := CreateTempDir(tempdir)
					if cache_err != nil {
						return cache_err
					}

					if FileExist(fmt.Sprintf("%s/etc/group", con.RootPath)) {
						cerr := Rename(fmt.Sprintf("%s/etc/group", con.RootPath), fmt.Sprintf("%s/etc/group", cache_temp_dir))
						if cerr != nil {
							return cerr
						}
					}
					if FileExist(fmt.Sprintf("%s/etc/passwd", con.RootPath)) {
						cerr := Rename(fmt.Sprintf("%s/etc/passwd", con.RootPath), fmt.Sprintf("%s/etc/passwd", cache_temp_dir))
						if cerr != nil {
							return cerr
						}
					}
					if FolderExist(fmt.Sprintf("%s/tmp", con.RootPath)) {
						RemoveAll(fmt.Sprintf("%s/tmp", con.RootPath))
					}
					if FileExist(fmt.Sprintf("%s/.wh.tmp", con.RootPath)) {
						RemoveFile(fmt.Sprintf("%s/.wh.tmp", con.RootPath))
					}
					//remove data symlink
					if len(con.DataSyncMap) > 0 {
						for _, kv := range strings.Split(con.DataSyncMap, ";") {
							if len(kv) > 0 {
								v := strings.Split(kv, ":")
								if len(v) == 2 && len(v[1]) > 0 {
									s_link := fmt.Sprintf("%s%s", con.RootPath, v[1])
									os.RemoveAll(s_link)
								}
							}
						}
					}

					//remove apt cache
					if FolderExist(fmt.Sprintf("%s/var/lib/apt/lists", con.RootPath)) {
						RemoveAll(fmt.Sprintf("%s/var/lib/apt/lists", con.RootPath))
					}

					if FolderExist(fmt.Sprintf("%s/var/lib/dpkg", con.RootPath)) {
						RemoveAll(fmt.Sprintf("%s/var/lib/dpkg", con.RootPath))
					}

					//step 1: tar rw layer
					layers := strings.Split(con.Layers, ":")
					layers = layers[1:]
					layers_full_path := []string{con.RootPath}
					for _, layer := range layers {
						layers_full_path = append(layers_full_path, fmt.Sprintf("%s/%s", con.BaseLayerPath, layer))
					}
					fmt.Println("taring rw layers...")
					//get temp dir
					temp_dir, temp_err := CreateTempDir(tempdir)
					if temp_err != nil {
						return temp_err
					}
					//tar rw layer
					cerr := TarLayer(con.RootPath, temp_dir, con.Id, layers_full_path)
					if cerr != nil {
						return cerr
					}
					//step 2: calculate shasum value and move it to image folder
					rw_tar_path := fmt.Sprintf("%s/%s.tar.gz", temp_dir, con.Id)
					shasum, serr := Sha256file(rw_tar_path)
					if serr != nil {
						return serr
					}
					//image dir is LPMX/.lpmxdata/.image
					//moving layer tarball to image folder
					fmt.Println("renaming rw layer...")
					image_dir := fmt.Sprintf("%s/.image", filepath.Dir(con.BaseLayerPath))
					src_tar_path := rw_tar_path
					target_tar_path := fmt.Sprintf("%s/%s.tar.gz", image_dir, shasum)
					rerr := os.Rename(src_tar_path, target_tar_path)
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not rename(move): %s to %s", src_tar_path, target_tar_path))
						return cerr
					}

					//moving rw layer to base folder
					//here, target place has suffix of .tar.gz
					rerr = os.Rename(con.RootPath, fmt.Sprintf("%s/%s.tar.gz", con.BaseLayerPath, shasum))
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not rename(move): %s to %s", con.RootPath, fmt.Sprintf("%s/%s", con.BaseLayerPath, shasum)))
						return cerr
					}

					//create new symlink
					new_symlink_path := fmt.Sprintf("%s/%s.tar.gz", filepath.Dir(con.RootPath), shasum)
					old_symlink_path := fmt.Sprintf("%s/%s.tar.gz", con.BaseLayerPath, shasum)
					rerr = os.Symlink(old_symlink_path, new_symlink_path)
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not symlink: %s to %s", old_symlink_path, new_symlink_path))
						return cerr
					}

					//moving workspace and copying setting.yml to new place
					docker_path := filepath.Dir(con.BaseLayerPath)
					new_workspace_path := fmt.Sprintf("%s/%s/%s/workspace", docker_path, newname, newtag)
					if !FolderExist(new_workspace_path) {
						derr := os.MkdirAll(new_workspace_path, os.FileMode(FOLDER_MODE))
						if derr != nil {
							cerr := ErrNew(derr, fmt.Sprintf("could not make dir %s", new_workspace_path))
							return cerr
						}
					}

					old_workspace_path := filepath.Dir(con.ConfigPath)
					rerr = os.Rename(old_workspace_path, fmt.Sprintf("%s/%s", new_workspace_path, id))
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not rename(move): %s to %s", old_workspace_path, fmt.Sprintf("%s/%s", new_workspace_path, id)))
						return cerr
					}
					//copy setting.yml rather than rename
					new_setting_path := fmt.Sprintf("%s/setting.yml", filepath.Dir(new_workspace_path))
					_, cerr = CopyFile(con.SettingPath, new_setting_path)
					if cerr != nil {
						return cerr
					}
					con.RootPath = fmt.Sprintf("%s/%s/rw", new_workspace_path, id)
					con.SettingPath = new_setting_path
					con.ConfigPath = fmt.Sprintf("%s/%s/.lpmx", new_workspace_path, id)
					con.LogPath = fmt.Sprintf("%s/log", con.ConfigPath)
					con.PatchedELFLoader = fmt.Sprintf("%s/%s/ld.so.patch", new_workspace_path, id)

					//step 3: froze rw layer and create new rw layer
					fmt.Println("cleaning up...")
					derr := os.Mkdir(con.RootPath, os.FileMode(FOLDER_MODE))
					if derr != nil {
						cerr := ErrNew(derr, fmt.Sprintf("could not make new folder: %s", con.RootPath))
						return cerr
					}
					//create new data sync folder
					for _, kv := range strings.Split(con.DataSyncMap, ";") {
						if len(kv) > 0 {
							v := strings.Split(kv, ":")
							derr := os.Symlink(v[0], fmt.Sprintf("%s%s", con.RootPath, v[1]))
							if derr != nil {
								cerr := ErrNew(derr, fmt.Sprintf("could not symlink, oldpath: %s, newpath: %s", v[0], v[1]))
								return cerr
							}
						}
					}
					//moving folders back to new rw folder
					if FileExist(fmt.Sprintf("%s/etc/group", cache_temp_dir)) {
						cerr := Rename(fmt.Sprintf("%s/etc/group", cache_temp_dir), fmt.Sprintf("%s/etc/group", con.RootPath))
						if cerr != nil {
							return cerr
						}
					}
					if FileExist(fmt.Sprintf("%s/etc/passwd", cache_temp_dir)) {
						cerr := Rename(fmt.Sprintf("%s/etc/passwd", cache_temp_dir), fmt.Sprintf("%s/etc/passwd", con.RootPath))
						if cerr != nil {
							return cerr
						}
					}
					//create new tmp
					os.MkdirAll(fmt.Sprintf("%s/tmp", con.RootPath), os.FileMode(FOLDER_MODE))
					f, _ := os.Create(fmt.Sprintf("%s/.wh.tmp", con.RootPath))
					f.Close()

					//step 4: modify container info
					new_layers := []string{"rw", fmt.Sprintf("%s.tar.gz", shasum)}
					new_layers = append(new_layers, strings.Split(con.Layers, ":")[1:]...)
					con.Layers = strings.Join(new_layers, ":")

					data, _ := StructMarshal(&con)
					LOGGER.WithFields(logrus.Fields{
						"con": con,
					}).Debug("DockerCommit update container info")
					cerr = WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))
					if cerr != nil {
						return cerr
					}
					old_imagebase := con.ImageBase
					con.ImageBase = fmt.Sprintf("%s:%s", newname, newtag)
					//update $/.lpmxsys/.info
					con.appendToSys()
					//end of updating container info

					//located inside $/.lpmxdata
					//start updating image info
					fmt.Println("updating image info...")
					mdata := make(map[string]interface{})
					mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", docker_path, newname, newtag)
					mdata["config"] = con.SettingPath
					mdata["image"] = fmt.Sprintf("%s/.image", docker_path)
					//get old layer map to update new image info
					image_map := doc.Images[old_imagebase]
					if image_map != nil {
						if map_interface, map_ok := image_map.(map[string]interface{}); map_ok {
							if old_map, old_ok := (map_interface["layer"]).(map[string]interface{}); old_ok {
								size, serr := GetFileSize(target_tar_path)
								if serr != nil {
									return serr
								}
								//here we clone the original map
								clone_map := CopyMap(old_map)
								clone_map[fmt.Sprintf("%s/%s.tar.gz", mdata["image"].(string), shasum)] = size
								mdata["layer"] = clone_map
								mdata["layer_order"] = fmt.Sprintf("%s:%s", map_interface["layer_order"].(string), fmt.Sprintf("%s/%s.tar.gz", mdata["image"].(string), shasum))

								//check if orig_layer_order exists
								if val, ok := map_interface["orig_layer_order"]; ok {
									mdata["orig_layer_order"] = fmt.Sprintf("%s:%s", val.(string), fmt.Sprintf("%s/%s.tar.gz", mdata["image"].(string), shasum))
								}
							} else {
								cerr := ErrNew(ErrType, fmt.Sprintf("doc.Image.Layer type is not right, actual: %T, want: map[string]interface{}", map_interface))
								return cerr
							}
						} else {
							cerr := ErrNew(ErrType, fmt.Sprintf("doc.Image type is not right, actual: %T, want: map[string]interface{}", image_map))
							return cerr
						}
					}
					mdata["workspace"] = fmt.Sprintf("%s/workspace", mdata["rootdir"])
					mdata["base"] = fmt.Sprintf("%s/.base", docker_path)
					mdata["imagetype"] = "Docker"

					doc.Images[fmt.Sprintf("%s:%s", newname, newtag)] = mdata
					mddata, _ := StructMarshal(doc)
					LOGGER.WithFields(logrus.Fields{
						"doc":        doc,
						"write_path": fmt.Sprintf("%s/.info", doc.RootDir),
					}).Debug("DockerCommit, update image info")
					cerr = WriteToFile(mddata, fmt.Sprintf("%s/.info", doc.RootDir))
					if cerr != nil {
						return cerr
					}

					//start adding docinfo
					var docinfo ImageInfo
					docinfo.Name = fmt.Sprintf("%s:%s", newname, newtag)
					docinfo.ImageType = "Docker"
					// layer_order is absolute path
					docinfo.LayersMap = make(map[string]int64)
					if mdata["layer"] != nil {
						layerinfo := mdata["layer"].(map[string]interface{})
						for key, value := range layerinfo {
							docinfo.LayersMap[path.Base(key)] = value.(int64)
						}
					}
					//here we remove the absolute path, only keep shasum value
					layersorder := strings.Split(mdata["layer_order"].(string), ":")
					for idx, l := range layersorder {
						layersorder[idx] = path.Base(l)
					}
					docinfo.Layers = strings.Join(layersorder, ":")

					LOGGER.WithFields(logrus.Fields{
						"docinfo":    docinfo,
						"write_path": fmt.Sprintf("%s/.info", mdata["rootdir"].(string)),
					}).Debug("DockerCommit, update docinfo info")
					dinfodata, _ := StructMarshal(docinfo)
					err = WriteToFile(dinfodata, fmt.Sprintf("%s/.info", mdata["rootdir"].(string)))
					if err != nil {
						return err
					}
					//end

					return nil
					//done
				} else {
					pid, _ := PidValue(pidfile)
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running with pid: %d, can't package layer, please stop it firstly", id, pid))
					return cerr
				}
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}
	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist", rootdir))
	}
	return err
}

func DockerMerge(name, user, pass string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	sysdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err != nil && err.Err != ErrNExist {
		return err
	}
	if err != nil && err.Err == ErrNExist {
		ret, err := MakeDir(rootdir)
		doc.RootDir = rootdir
		doc.Images = make(map[string]interface{})
		if !ret {
			return err
		}
	}
	if !strings.Contains(name, ":") {
		name = name + ":latest"
	}

	mdata := make(map[string]interface{})
	tdata := strings.Split(name, ":")
	tname := tdata[0]
	ttag := tdata[1]
	mdata["rootdir"] = fmt.Sprintf("%s/%s/%s-merge", doc.RootDir, tname, ttag)
	mdata["config"] = fmt.Sprintf("%s/setting.yml", mdata["rootdir"].(string))
	mdata["image"] = fmt.Sprintf("%s/.image", rootdir)
	image_dir, _ := mdata["image"].(string)
	//name is something like "ubuntu:16.04"
	var layer_order []string
	if mdata_orig, ok := doc.Images[name]; ok {
		if mdata_map, m_ok := mdata_orig.(map[string]interface{}); m_ok {
			layer_order = strings.Split(mdata_map["layer_order"].(string), ":")
		}
	} else {
		_, layer_order, err = DownloadLayers(user, pass, tname, ttag, image_dir)
		if err != nil {
			return err
		}
	}

	//base folder
	base := fmt.Sprintf("%s/.base", rootdir)
	if !FolderExist(base) {
		MakeDir(base)
	}
	mdata["base"] = base

	//workspace folder
	workspace := fmt.Sprintf("%s/workspace", mdata["rootdir"])
	if !FolderExist(workspace) {
		MakeDir(workspace)
	}
	mdata["workspace"] = workspace

	//create temp folder
	tmpdir, terr := CreateTempDir(mdata["base"].(string))
	//extract firstly
	for _, k := range layer_order {
		k = path.Base(k)
		tar_path := fmt.Sprintf("%s/%s", image_dir, k)
		err := UntarLayer(tar_path, tmpdir)
		if err != nil {
			return err
		}
	}

	//download setting from github
	rdir, _ := mdata["rootdir"].(string)

	yaml := fmt.Sprintf("%s/distro.management.yml", sysdir)
	err = DownloadFilefromGithubPlus(tname, ttag, "setting.yml", SETTING_URL, rdir, yaml)
	if err != nil {
		LOGGER.WithFields(logrus.Fields{
			"err":    err,
			"toPath": rdir,
		}).Error("Download setting from github failure and could not rollback to default one")
		return err
	}

	//tar folder to tarball
	LOGGER.Info("Start merging layers...please wait")
	target_name := RandomString(10)
	terr = TarLayer(tmpdir, image_dir, target_name, nil)
	if terr != nil {
		return terr
	}

	tar_image_path := fmt.Sprintf("%s/%s.tar.gz", image_dir, target_name)
	sha256, serr := Sha256file(tar_image_path)
	if serr != nil {
		return serr
	}

	//rename folder and file
	new_folder_name := fmt.Sprintf("%s/%s.tar.gz", mdata["base"].(string), sha256)
	new_image_name := fmt.Sprintf("%s/%s.tar.gz", mdata["image"].(string), sha256)
	rerr := Rename(tmpdir, new_folder_name)
	if rerr != nil {
		return rerr
	}
	rerr = Rename(tar_image_path, new_image_name)
	if rerr != nil {
		return rerr
	}

	//write info to disk file
	ret := make(map[string]int64)
	size, ierr := GetFileLength(new_image_name)
	if ierr != nil {
		return ierr
	}
	//new layer name : size
	ret[new_image_name] = size
	mdata["layer"] = ret
	//20200204 here we add backup of original layers order info in order for later package command

	mdata["orig_layer_order"] = strings.Join(layer_order, ":")

	//then we set new layer_order
	mdata["layer_order"] = new_image_name

	//add docker info file(.info)
	if !FolderExist(mdata["rootdir"].(string)) {
		merr := os.MkdirAll(mdata["rootdir"].(string), os.FileMode(FOLDER_MODE))
		if merr != nil {
			cerr := ErrNew(merr, fmt.Sprintf("could not mkdir %s", mdata["rootdir"].(string)))
			return cerr
		}
	}

	var docinfo ImageInfo
	docinfo.Name = name
	docinfo.ImageType = "Docker"
	layersmap := make(map[string]int64)
	//sha256:size
	for k, v := range ret {
		layersmap[path.Base(k)] = v
	}
	docinfo.LayersMap = layersmap
	docinfo.Layers = sha256

	LOGGER.WithFields(logrus.Fields{
		"doc": docinfo,
	}).Debug("DockerMerge, update image info")
	dinfodata, _ := StructMarshal(docinfo)
	err = WriteToFile(dinfodata, fmt.Sprintf("%s/.info", mdata["rootdir"].(string)))
	if err != nil {
		return err
	}

	//add map to this image
	//change name here
	new_name := fmt.Sprintf("%s-merge", name)
	doc.Images[new_name] = mdata
	LOGGER.WithFields(logrus.Fields{
		"docinfo": doc,
	}).Debug("DockerMerge, update docinfo info")
	ddata, _ := StructMarshal(doc)
	err = WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
	if err != nil {
		return err
	}

	return nil
}

func SingularityLoad(file string, name string, tag string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	sysdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	tempdir := fmt.Sprintf("%s/.temp", currdir)
	//we delete temp dir if it exists at the end of the function
	defer func() {
		if FolderExist(tempdir) {
			os.RemoveAll(tempdir)
		}
	}()

	//check if file exists
	if !FileExist(file) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does exist", file))
		return cerr
	}

	//configure Singularity related info locally
	var sig Image
	err = unmarshalObj(rootdir, &sig)

	if err != nil && err.Err == ErrNExist {
		ret, err := MakeDir(rootdir)
		sig.RootDir = rootdir
		sig.Images = make(map[string]interface{})
		if !ret {
			return err
		}
	}

	//get temp folder for extraction
	tmpdir, terr := CreateTempDir(tempdir)
	if terr != nil {
		return terr
	}
	image_dir := fmt.Sprintf("%s/.image", rootdir)
	full_name := fmt.Sprintf("%s:%s", name, tag)

	//step 1: extract squashfs from sif file
	rand_id := RandomString(10)
	err = ExtractSquashfs(file, fmt.Sprintf("%s/%s", tmpdir, rand_id))
	if err != nil {
		return err
	}

	//step 2: move it to image folder
	ret, layer_order, lerr := LoadSquashfs(fmt.Sprintf("%s/%s", tmpdir, rand_id), image_dir)
	if lerr != nil {
		return lerr
	}

	//step 3: generate necessary info
	if _, ok := sig.Images[full_name]; ok {
		cerr := ErrNew(ErrExist, fmt.Sprintf("%s already exists", full_name))
		return cerr
	} else {
		tname := name
		ttag := tag
		mdata := make(map[string]interface{})
		mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", sig.RootDir, tname, ttag)
		mdata["config"] = fmt.Sprintf("%s/setting.yml", mdata["rootdir"].(string))
		mdata["image"] = fmt.Sprintf("%s/.image", rootdir)
		mdata["layer"] = ret
		mdata["layer_order"] = strings.Join(layer_order, ":")
		mdata["imagetype"] = "Singularity"

		//add image info file(.info)
		if !FolderExist(mdata["rootdir"].(string)) {
			merr := os.MkdirAll(mdata["rootdir"].(string), os.FileMode(FOLDER_MODE))
			if merr != nil {
				cerr := ErrNew(merr, fmt.Sprintf("could not mkdir %s", mdata["rootdir"].(string)))
				return cerr
			}
		}
		var siginfo ImageInfo
		siginfo.Name = full_name
		siginfo.ImageType = "Sigularity"
		// layer_order is absolute path
		//siginfo layers map should remove absolute path of host
		layersmap := make(map[string]int64)
		for k, v := range ret {
			layersmap[path.Base(k)] = v
		}
		siginfo.LayersMap = layersmap

		var layer_sha []string
		for _, layer := range layer_order {
			layer_sha = append(layer_sha, path.Base(layer))
		}
		siginfo.Layers = strings.Join(layer_sha, ":")

		LOGGER.WithFields(logrus.Fields{
			"siginfo": siginfo,
		}).Debug("SingularityLoad debug, siginfo debug")

		dinfodata, _ := StructMarshal(siginfo)
		err = WriteToFile(dinfodata, fmt.Sprintf("%s/.info", mdata["rootdir"].(string)))
		if err != nil {
			return err
		}
		//end

		workspace := fmt.Sprintf("%s/workspace", mdata["rootdir"])
		if !FolderExist(workspace) {
			MakeDir(workspace)
		}
		mdata["workspace"] = workspace

		//extract layers
		base := fmt.Sprintf("%s/.base", rootdir)
		if !FolderExist(base) {
			MakeDir(base)
		}
		mdata["base"] = base

		for _, k := range layer_order {
			k = path.Base(k)
			tar_path := fmt.Sprintf("%s/%s", image_dir, k)
			layerfolder := fmt.Sprintf("%s/%s", mdata["base"], k)
			if !FolderExist(layerfolder) {
				MakeDir(layerfolder)
			}

			err := Unsquashfs(tar_path, layerfolder)
			if err != nil {
				return err
			}
		}

		//download setting from github
		rdir, _ := mdata["rootdir"].(string)

		yaml := fmt.Sprintf("%s/distro.management.yml", sysdir)
		err = DownloadFilefromGithubPlus(tname, ttag, "setting.yml", SETTING_URL, rdir, yaml)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download setting from github failure and could not rollback to default one")
			return err
		}

		//add map to this image
		sig.Images[full_name] = mdata

		ddata, _ := StructMarshal(sig)
		err = WriteToFile(ddata, fmt.Sprintf("%s/.info", sig.RootDir))
		if err != nil {
			return err
		}
		return nil
	}
}

func DockerLoad(file string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	sysdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	tempdir := fmt.Sprintf("%s/.temp", currdir)
	//we delete temp dir if it exists at the end of the function
	defer func() {
		if FolderExist(tempdir) {
			os.RemoveAll(tempdir)
		}
	}()

	//check if file exists
	if !FileExist(file) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does exist", file))
		return cerr
	}

	//configure Docker related info locally
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err != nil && err.Err != ErrNExist {
		return err
	}

	if err != nil && err.Err == ErrNExist {
		ret, err := MakeDir(rootdir)
		doc.RootDir = rootdir
		doc.Images = make(map[string]interface{})
		if !ret {
			return err
		}
	}

	//get temp folder for extraction
	tmpdir, terr := CreateTempDir(tempdir)
	if terr != nil {
		return terr
	}

	//untar tar ball
	uerr := Untar(file, tmpdir)
	if uerr != nil {
		return uerr
	}

	image_dir := fmt.Sprintf("%s/.image", rootdir)
	name, ret, layer_order, lerr := LoadDockerTar(tmpdir, image_dir)
	if lerr != nil {
		return lerr
	}
	if _, ok := doc.Images[name]; ok {
		cerr := ErrNew(ErrExist, fmt.Sprintf("%s already exists", name))
		return cerr
	} else {
		tdata := strings.Split(name, ":")
		tname := tdata[0]
		ttag := tdata[1]
		mdata := make(map[string]interface{})
		mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", doc.RootDir, tname, ttag)
		mdata["config"] = fmt.Sprintf("%s/setting.yml", mdata["rootdir"].(string))
		mdata["image"] = fmt.Sprintf("%s/.image", rootdir)
		mdata["layer"] = ret
		mdata["layer_order"] = strings.Join(layer_order, ":")
		mdata["imagetype"] = "Docker"

		//add docker info file(.info)
		if !FolderExist(mdata["rootdir"].(string)) {
			merr := os.MkdirAll(mdata["rootdir"].(string), os.FileMode(FOLDER_MODE))
			if merr != nil {
				cerr := ErrNew(merr, fmt.Sprintf("could not mkdir %s", mdata["rootdir"].(string)))
				return cerr
			}
		}
		var docinfo ImageInfo
		docinfo.Name = name
		docinfo.ImageType = "Docker"
		// layer_order is absolute path
		//docinfo layers map should remove absolute path of host
		layersmap := make(map[string]int64)
		for k, v := range ret {
			layersmap[path.Base(k)] = v
		}
		docinfo.LayersMap = layersmap

		var layer_sha []string
		for _, layer := range layer_order {
			layer_sha = append(layer_sha, path.Base(layer))
		}
		docinfo.Layers = strings.Join(layer_sha, ":")

		LOGGER.WithFields(logrus.Fields{
			"docinfo": docinfo,
		}).Debug("DockerLoad debug, docinfo debug")

		dinfodata, _ := StructMarshal(docinfo)
		err = WriteToFile(dinfodata, fmt.Sprintf("%s/.info", mdata["rootdir"].(string)))
		if err != nil {
			return err
		}
		//end

		workspace := fmt.Sprintf("%s/workspace", mdata["rootdir"])
		if !FolderExist(workspace) {
			MakeDir(workspace)
		}
		mdata["workspace"] = workspace

		/**
		patchfolder := fmt.Sprintf("%s/patch", mdata["rootdir"])
		if !FolderExist(patchfolder) {
			MakeDir(patchfolder)
		}
		**/

		//extract layers
		base := fmt.Sprintf("%s/.base", rootdir)
		if !FolderExist(base) {
			MakeDir(base)
		}
		mdata["base"] = base

		for _, k := range layer_order {
			k = path.Base(k)
			tar_path := fmt.Sprintf("%s/%s", image_dir, k)
			layerfolder := fmt.Sprintf("%s/%s", mdata["base"], k)
			if !FolderExist(layerfolder) {
				MakeDir(layerfolder)
			}

			err := Untar(tar_path, layerfolder)
			if err != nil {
				return err
			}
		}

		//download setting from github
		rdir, _ := mdata["rootdir"].(string)

		yaml := fmt.Sprintf("%s/distro.management.yml", sysdir)
		err = DownloadFilefromGithubPlus(tname, ttag, "setting.yml", SETTING_URL, rdir, yaml)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download setting from github failure and could not rollback to default one")
			return err
		}

		/**
		//here we start downloading patch tar ball to rdir/patch folder
		pdir := fmt.Sprintf("%s/patch", rdir)
		//save patch.tar.gz into rdir and untar it to pdir
		err = DownloadFilefromGithubPlus(tname, ttag, "patch.tar.gz", SETTING_URL, rdir, yaml)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download patch.tar.gz from github failure and could not rollback to default one")
			return err
		} else {
			//if download success, we have to untar it
			err = Untar(fmt.Sprintf("%s/patch.tar.gz", rdir), pdir)
			if err != nil {
				return err
			}
		}
		**/

		//add map to this image
		doc.Images[name] = mdata

		ddata, _ := StructMarshal(doc)
		err = WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
		if err != nil {
			return err
		}
	}
	return nil
}

func DockerDownload(name string, user string, pass string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	sysdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err != nil && err.Err != ErrNExist {
		return err
	}
	if err != nil && err.Err == ErrNExist {
		ret, err := MakeDir(rootdir)
		doc.RootDir = rootdir
		doc.Images = make(map[string]interface{})
		if !ret {
			return err
		}
	}
	if !strings.Contains(name, ":") {
		name = name + ":latest"
	}
	if _, ok := doc.Images[name]; ok {
		cerr := ErrNew(ErrExist, fmt.Sprintf("%s already exists", name))
		return cerr
	} else {
		//downloading image from github
		tdata := strings.Split(name, ":")
		tname := tdata[0]
		ttag := tdata[1]
		mdata := make(map[string]interface{})
		mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", doc.RootDir, tname, ttag)
		mdata["config"] = fmt.Sprintf("%s/setting.yml", mdata["rootdir"].(string))
		mdata["image"] = fmt.Sprintf("%s/.image", rootdir)
		image_dir, _ := mdata["image"].(string)

		//download layers
		ret, layer_order, err := DownloadLayers(user, pass, tname, ttag, image_dir)
		if err != nil {
			return err
		}
		mdata["layer"] = ret
		mdata["layer_order"] = strings.Join(layer_order, ":")
		mdata["imagetype"] = "Docker"

		//add docker info file(.info)
		if !FolderExist(mdata["rootdir"].(string)) {
			merr := os.MkdirAll(mdata["rootdir"].(string), os.FileMode(FOLDER_MODE))
			if merr != nil {
				cerr := ErrNew(merr, fmt.Sprintf("could not mkdir %s", mdata["rootdir"].(string)))
				return cerr
			}
		}
		var docinfo ImageInfo
		docinfo.Name = name
		docinfo.ImageType = "Docker"
		// layer_order is absolute path
		//docinfo layers map should remove absolute path of host
		layersmap := make(map[string]int64)
		for k, v := range ret {
			layersmap[path.Base(k)] = v
		}
		docinfo.LayersMap = layersmap

		var layer_sha []string
		for _, layer := range layer_order {
			layer_sha = append(layer_sha, path.Base(layer))
		}
		docinfo.Layers = strings.Join(layer_sha, ":")
		LOGGER.WithFields(logrus.Fields{
			"docinfo": docinfo,
		}).Debug("DockerDownload debug, add docinfo")

		dinfodata, _ := StructMarshal(docinfo)
		err = WriteToFile(dinfodata, fmt.Sprintf("%s/.info", mdata["rootdir"].(string)))
		if err != nil {
			return err
		}
		//end

		workspace := fmt.Sprintf("%s/workspace", mdata["rootdir"])
		if !FolderExist(workspace) {
			MakeDir(workspace)
		}
		mdata["workspace"] = workspace

		/**
		patchfolder := fmt.Sprintf("%s/patch", mdata["rootdir"])
		if !FolderExist(patchfolder) {
			MakeDir(patchfolder)
		}
		**/

		//extract layers
		base := fmt.Sprintf("%s/.base", rootdir)
		if !FolderExist(base) {
			MakeDir(base)
		}
		mdata["base"] = base

		for _, k := range layer_order {
			k = path.Base(k)
			tar_path := fmt.Sprintf("%s/%s", image_dir, k)
			layerfolder := fmt.Sprintf("%s/%s", mdata["base"], k)
			if !FolderExist(layerfolder) {
				MakeDir(layerfolder)
			}

			err := Untar(tar_path, layerfolder)
			if err != nil {
				return err
			}
		}

		//download setting from github
		rdir, _ := mdata["rootdir"].(string)

		yaml := fmt.Sprintf("%s/distro.management.yml", sysdir)
		err = DownloadFilefromGithubPlus(tname, ttag, "setting.yml", SETTING_URL, rdir, yaml)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download setting from github failure and could not rollback to default one")
			return err
		}

		//20200120 disable downloading patched files, because we do not need it any longer
		/**
		//here we start downloading patch tar ball to rdir/patch folder
		pdir := fmt.Sprintf("%s/patch", rdir)
		//save patch.tar.gz into rdir and untar it to pdir
		err = DownloadFilefromGithubPlus(tname, ttag, "patch.tar.gz", SETTING_URL, rdir, yaml)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download patch.tar.gz from github failure and could not rollback to default one")
			//return err

			//if we could not download patch.tar.gz from github, we have to patch ld.so inside layers
			//
		} else {
			//if download success, we have to untar it
			err = Untar(fmt.Sprintf("%s/patch.tar.gz", rdir), pdir)
			if err != nil {
				return err
			}
		}
		**/

		//add map to this image
		doc.Images[name] = mdata

		LOGGER.WithFields(logrus.Fields{
			"doc": doc,
		}).Debug("DockerDownload debug, add image info to global images")
		ddata, _ := StructMarshal(doc)
		err = WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
		if err != nil {
			return err
		}
		return nil
	}
}

func DockerPush(user string, pass string, name string, tag string, id string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	//first check whether the container is running
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)
	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				config_path := val["ConfigPath"].(string)

				var con Container
				err = unmarshalObj(config_path, &con)
				if err != nil {
					return err
				}
				pidfile := fmt.Sprintf("%s/container.pid", path.Dir(con.RootPath))

				if pok, _ := PidIsActive(pidfile); !pok {
					//parameters are src/target folder, target file name, and layer paths
					//step 1: tar rw layer
					layers := strings.Split(con.Layers, ":")
					layers = layers[1:]
					layers_full_path := []string{con.RootPath}
					for _, layer := range layers {
						layers_full_path = append(layers_full_path, fmt.Sprintf("%s/%s", con.BaseLayerPath, layer))
					}
					fmt.Println("taring rw layers...")
					cerr := TarLayer(con.RootPath, "/tmp", con.Id, layers_full_path)
					if cerr != nil {
						return cerr
					}
					//step 2: upload this tar ball to docker hub and backup inside lpmx
					fmt.Println("uploading layers...")
					shasum, cerr := UploadLayers(user, pass, name, tag, fmt.Sprintf("/tmp/%s.tar.gz", con.Id), con.ImageBase)
					if cerr != nil {
						return cerr
					}
					image_dir := fmt.Sprintf("%s/image", filepath.Dir(con.BaseLayerPath))
					src_tar_path := fmt.Sprintf("/tmp/%s.tar.gz", con.Id)
					target_tar_path := fmt.Sprintf("%s/%s", image_dir, shasum)
					err := os.Rename(src_tar_path, target_tar_path)
					if err != nil {
						cerr := ErrNew(err, fmt.Sprintf("could not rename(move): %s to %s", src_tar_path, target_tar_path))
						return cerr
					}

					err = os.Rename(con.RootPath, fmt.Sprintf("%s/%s", con.BaseLayerPath, shasum))
					if err != nil {
						cerr := ErrNew(err, fmt.Sprintf("could not rename(move): %s to %s", con.RootPath, fmt.Sprintf("%s/%s", con.BaseLayerPath, shasum)))
						return cerr
					}
					//step 3: froze rw layer and create new rw layer
					fmt.Println("cleaning up...")
					err = os.Mkdir(con.RootPath, os.FileMode(FOLDER_MODE))
					if err != nil {
						cerr := ErrNew(err, fmt.Sprintf("could not make new folder: %s", con.RootPath))
						return cerr
					}
					//step 4: modify container info
					new_layers := []string{"rw", shasum}
					new_layers = append(new_layers, layers...)
					con.Layers = strings.Join(new_layers, ":")

					data, _ := StructMarshal(&con)
					cerr = WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))
					if cerr != nil {
						return cerr
					}
					con.ImageBase = fmt.Sprintf("%s:%s", name, tag)
					con.appendToSys()
					//done
				} else {
					pid, _ := PidValue(pidfile)
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running with pid: %d, can't package layer, please stop it firstly", id, pid))
					return cerr
				}
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}
	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err
}

func CommonList(imagetype string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	fmt.Println(fmt.Sprintf("%s", "Name"))
	if err == nil {
		for k, v := range doc.Images {
			if vval, vok := v.(map[string]interface{}); vok {
				if strings.Compare(vval["imagetype"].(string), imagetype) == 0 {
					fmt.Println(fmt.Sprintf("%s", k))
				}
			}
		}
		return nil
	}
	if err.Err != ErrNExist {
		return err
	}
	return nil
}

func DockerReset(name string) *Error {
	if !strings.Contains(name, ":") {
		name = name + ":latest"
	}
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err == nil {
		if name_data, name_ok := doc.Images[name].(map[string]interface{}); name_ok {
			image_dir, _ := name_data["image"].(string)
			layer_order := name_data["layer_order"].(string)
			for _, k := range strings.Split(layer_order, ":") {
				k = path.Base(k)
				tar_path := fmt.Sprintf("%s/%s", image_dir, k)
				layerfolder := fmt.Sprintf("%s/%s", name_data["base"].(string), k)
				LOGGER.WithFields(logrus.Fields{
					"tar_path":     tar_path,
					"layer_folder": layerfolder,
				}).Debug("docker reset images, reextract tar ball")
				if FolderExist(layerfolder) {
					RemoveAll(layerfolder)
				}

				if !FolderExist(layerfolder) {
					MakeDir(layerfolder)
				}

				err := Untar(tar_path, layerfolder)
				if err != nil {
					return err
				}
			}
			return nil
		}
		cerr := ErrNew(ErrNExist, fmt.Sprintf("name: %s is not found", name))
		return cerr
	}
	return err
}

func CommonFastRun(name, volume_map, command, engine, execmaps string) *Error {
	configmap, err := generateContainer(name, "", volume_map, engine)
	if err != nil {
		return err
	}
	id := (*configmap)["id"].(string)
	if len(execmaps) > 0 {
		(*configmap)["execmaps"] = execmaps
	}
	err = Run(configmap, command)
	//remove container
	if err != nil {
		err = Destroy(id)
		return err
	}
	err = Destroy(id)
	return err
}

//create container based on images
func CommonCreate(name, container_name, volume_map, engine, execmaps string) *Error {
	configmap, err := generateContainer(name, container_name, volume_map, engine)
	if err != nil {
		return err
	}
	if len(execmaps) > 0 {
		(*configmap)["execmaps"] = execmaps
	}
	err = Run(configmap)
	return err
}

//delete image
func CommonDelete(name string, permernant bool) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	//check if there are containers assocated with current image
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)
	if err == nil {
		//range containers
		for key, value := range sys.Containers {
			if vval, vok := value.(map[string]interface{}); vok {
				config_path := vval["ConfigPath"].(string)
				var con Container
				err = unmarshalObj(config_path, &con)
				if err != nil {
					return err
				}

				if con.ImageBase == name {
					cerr := ErrNew(ErrOperation, fmt.Sprintf("container: %s still relies on image: %s", key, name))
					return cerr
				}
			} else {
				cerr := ErrNew(ErrType, "container type is not map[string]interface{}")
				return cerr
			}
		}
	} else {
		return err
	}

	rootdir = fmt.Sprintf("%s/.lpmxdata", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err == nil {
		if val, ok := doc.Images[name]; ok {
			if vval, vok := val.(map[string]interface{}); vok {
				if permernant {
					//we need to delete image files and folder info
					image_dir := vval["image"].(string)
					base_dir := vval["base"].(string)
					layer_order := vval["layer_order"].(string)
					for _, layer := range strings.Split(layer_order, ":") {
						layer_name := filepath.Base(layer)
						LOGGER.WithFields(logrus.Fields{
							"folder to delete": fmt.Sprintf("%s/%s", base_dir, layer_name),
							"file to delete":   fmt.Sprintf("%s/%s", image_dir, layer_name),
						}).Debug("Docker delete info")
						_, rerr := RemoveAll(fmt.Sprintf("%s/%s", image_dir, layer_name))
						if rerr != nil {
							return rerr
						}
						_, rerr = RemoveAll(fmt.Sprintf("%s/%s", base_dir, layer_name))
						if rerr != nil {
							return rerr
						}
					}
				}
				dir, _ := vval["rootdir"].(string)
				rok, rerr := RemoveAll(dir)
				if rok {
					delete(doc.Images, name)
					ddata, _ := StructMarshal(doc)
					err = WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
					if err != nil {
						return err
					}
					return nil
				} else {
					return rerr
				}
			}
		}
		cerr := ErrNew(ErrType, "doc.Images type error")
		return cerr
	}
	return err
}

func Expose(id string, ipath string, name string) *Error {
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				var con Container
				info := fmt.Sprintf("%s/.lpmx/.info", filepath.Dir(val["RootPath"].(string)))
				if FileExist(info) {
					data, err := ReadFromFile(info)
					if err == nil {
						err := StructUnmarshal(data, &con)
						if err != nil {
							return err
						}
						if !strings.Contains(con.ExposeExe, ipath) {
							if con.ExposeExe == "" {
								con.ExposeExe = ipath
							} else {
								con.ExposeExe = fmt.Sprintf("%s:%s", con.ExposeExe, ipath)
							}
						}

						bindir := fmt.Sprintf("%s/bin", currdir)
						if !FolderExist(bindir) {
							_, err := MakeDir(bindir)
							if err != nil {
								return err
							}
						}

						bname := name
						bdir := fmt.Sprintf("%s/%s", bindir, bname)
						if FileExist(bdir) {
							RemoveFile(bdir)
						}
						f, ferr := os.OpenFile(bdir, os.O_RDWR|os.O_CREATE, 0755)
						if ferr != nil {
							cerr := ErrNew(ferr, fmt.Sprintf("can not create exposed file %s", bdir))
							return cerr
						}

						ppath := fmt.Sprintf("%s/%s", currdir, os.Args[0])
						ppath = path.Clean(ppath)
						code := "#!/bin/bash\n" + ppath +
							" resume " + id + " \"" + ipath + " " + "$@\"" +
							"\n"

						fmt.Fprintf(f, code)
						defer f.Close()

						//write back
						data, _ := StructMarshal(&con)
						WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))
					}
				} else {
					cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.info doesn't exist", val["RootPath"].(string)))
					return cerr
				}
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}

	if err == ErrNExist {
		err.AddMsg(fmt.Sprintf("%s does not exist, you may need to use 'lpmx init' firstly", rootdir))
	}
	return err
}

/**
container methods
**/

func (con *Container) setupContainer() *Error {
	_, err := MakeDir(con.ConfigPath)
	if err != nil {
		return err
	}
	err = con.createContainer()
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
	return nil
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
			return cerr
		}
	}
	return nil
}

func (con *Container) genEnv(envmap map[string]string) (map[string]string, *Error) {
	env := make(map[string]string)
	env["ContainerId"] = con.Id
	env["ContainerRoot"] = con.RootPath
	env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so %s/libfakeroot.so", con.SysDir, con.SysDir)
	env["MEMCACHED_PID"] = con.MemcachedServerList[0]
	env["TERM"] = "xterm"
	env["SHELL"] = con.UserShell
	env["ContainerLayers"] = con.Layers
	env["ContainerBasePath"] = con.BaseLayerPath
	env["FAKECHROOT_ELFLOADER"] = con.PatchedELFLoader
	env["PWD"] = "/"
	env["HOME"] = "/root"
	env["FAKED_MODE"] = "unknown-is-root"
	env["BaseType"] = con.BaseType
	env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	env["ContainerConfigPath"] = con.ConfigPath
	if len(con.Execmaps) > 0 {
		env["FAKECHROOT_EXEC_SWITCH"] = "true"
	}

	//set default LD_LIBRARY_LPMX
	var libs []string
	//add libmemcached and other libs
	currdir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	libs = append(libs, fmt.Sprintf("%s/.lpmxsys", currdir))

	/**
	//20191230 we append patch folder path in a seperated env var indicating that patched existing ld.so related stuff
	patchfolder := fmt.Sprintf("%s/patch", filepath.Dir(filepath.Dir(filepath.Dir(con.RootPath))))
	env["FAKECHROOT_LDPatchPath"] = patchfolder
	**/

	//20200115 we append fakechroot system library folder in a seperated env var
	env["FAKECHROOT_SyslibPath"] = fmt.Sprintf("%s/.lpmxsys", currdir)

	//******* important, here we do not use LD_LIBRARY_LPMX any longer, as we will directly use LD_LIBRARY_PATH inside container, and make fakechroot to generate LD_LIBRARY_PATH itself base on layers info.

	//find from base layers
	//for _, v := range LD_LIBRARY_PATH_DEFAULT {
	//	lib_paths, err := GuessPathsContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v, false)
	//	if err != nil {
	//		continue
	//	} else {
	//		libs = append(libs, lib_paths...)
	//	}
	//}

	//add current rw layer
	//for _, v := range LD_LIBRARY_PATH_DEFAULT {
	//	libs = append(libs, fmt.Sprintf("%s/%s", con.RootPath, v))
	//}

	//set default FAKECHROOT_EXCLUDE_PATH
	env["FAKECHROOT_EXCLUDE_PATH"] = "/dev:/proc:/sys"

	//set data sync folder
	env["FAKECHROOT_DATA_SYNC"] = con.DataSyncFolder

	//set default FAKECHROOT_CMD_SUBSET
	env["FAKECHROOT_CMD_SUBST"] = "/sbin/ldconfig.real=/bin/true:/sbin/insserv=/bin/true:/sbin/ldconfig=/bin/true:/usr/bin/ischroot=/bin/true:/usr/bin/mkfifo=/bin/true"

	//pass current executable to libfakechroot.so so that when external exe are triggered, they might need the current executable location
	exe_path, _ := os.Executable()
	env["LPMX_EXECUTABLE"] = exe_path

	//export env
	if data, data_ok := con.SettingConf["export_env"]; data_ok {
		if d1, o1 := data.([]interface{}); o1 {
			for _, d1_1 := range d1 {
				if d1_11, o1_11 := d1_1.(interface{}); o1_11 {
					switch d1_11.(type) {
					case map[interface{}]interface{}:
						for k, v := range d1_11.(map[interface{}]interface{}) {
							if v1, vo1 := v.(string); vo1 {
								if k1, ko1 := k.(string); ko1 {
									var err *Error
									env[k1], err = GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v1, false)
									LOGGER.WithFields(logrus.Fields{
										"k":   k1,
										"v":   v1,
										"err": err,
									}).Debug("export_env, map[interface{}]interface, string")
									if err != nil {
										continue
									}
								}
							}
							if v1, vo1 := v.([]interface{}); vo1 {
								var libs []string
								for _, vv1 := range v1 {
									vv1_abs, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), vv1.(string), false)
									if err != nil {
										continue
									}
									libs = append(libs, vv1_abs)
								}
								if k1, ok1 := k.(string); ok1 {
									env[k1] = strings.Join(libs, ":")
								}
								LOGGER.WithFields(logrus.Fields{
									"k": k.(string),
									"v": libs,
								}).Debug("export_env, map[interface{}]interface, string array")
							}
						}
					case map[string]interface{}:
						for k, v := range d1_11.(map[string]interface{}) {
							if v1, vo1 := v.(string); vo1 {
								var err *Error
								env[k], err = GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v1, false)
								LOGGER.WithFields(logrus.Fields{
									"k":   k,
									"v":   v1,
									"err": err,
								}).Debug("export_env, map[string]interface, string")
								if err != nil {
									continue
								}
							}
							if v1, vo1 := v.([]interface{}); vo1 {
								var libs []string
								for _, vv1 := range v1 {
									vv1_abs, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), vv1.(string), false)
									if err != nil {
										continue
									}
									libs = append(libs, vv1_abs)
								}
								env[k] = strings.Join(libs, ":")
								LOGGER.WithFields(logrus.Fields{
									"k": k,
									"v": libs,
								}).Debug("export_env, map[string]interface, string array")
							}
						}
					}

				}
			}
		}

	}

	LOGGER.WithFields(logrus.Fields{
		"allenv": con.SettingConf,
	}).Debug("all env vars read from setting.yaml")

	if len(libs) > 0 {
		//if we have customized LD_LIBRARY_PATH set inside yaml file, let us merge them
		if ld_library_val, ld_library_ok := env["LD_LIBRARY_PATH"]; ld_library_ok {
			env["LD_LIBRARY_PATH"] = fmt.Sprintf("%s:%s", strings.Join(libs, ":"), ld_library_val)
		} else {
			env["LD_LIBRARY_PATH"] = strings.Join(libs, ":")
		}
	}

	if ldso_path, ldso_ok := con.SettingConf["fakechroot_elfloader"]; ldso_ok {
		elfloader_path, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), ldso_path.(string), true)
		if err != nil {
			return nil, err
		}
		env["FAKECHROOT_ELFLOADER"] = elfloader_path
	}

	if ep_path, ep_ok := con.SettingConf["fakechroot_exclude_ex_path"]; ep_ok {
		env["FAKECHROOT_EXCLUDE_EX_PATH"] = ep_path.(string)
	}

	//add all FAKECHROOT_ environment variables to LPMX
	for _, str := range os.Environ() {
		if strings.HasPrefix(str, "FAKECHROOT_") {
			values := strings.Split(str, "=")
			env[values[0]] = values[1]
		}
	}

	//write host env vars into the file
	env_path := fmt.Sprintf("%s/.env", con.ConfigPath)
	if FileExist(env_path) {
		RemoveFile(env_path)
	}
	envs := []string{"placer"}
	for _, str := range os.Environ() {
		if strings.Contains(str, "{") || strings.Contains(str, "}") || strings.Contains(str, "(") || strings.Contains(str, ")") || !strings.Contains(str, "=") {
			continue
		}
		envs = append(envs, str)
	}
	envs[0] = strconv.Itoa(len(envs) - 1)
	ofile, _ := os.Create(env_path)
	defer ofile.Close()
	writer := bufio.NewWriter(ofile)
	for _, line := range envs {
		fmt.Fprintln(writer, line)
	}
	writer.Flush()

	//process engine type
	if _, vok := envmap["engine"]; vok && con.Engine != "" {
		env["FAKECHROOT_ENGINE"] = "TRUE"
		env["FAKECHROOT_ENGINE_TYPE"] = con.Engine
		for _, str := range os.Environ() {
			if strings.HasPrefix(str, con.Engine) {
				values := strings.Split(str, "=")
				env[values[0]] = values[1]
			}
		}

		//add exclude paths
		paths := strings.Split(env["FAKECHROOT_EXCLUDE_PATH"], ":")
		if sk, sok := os.LookupEnv(fmt.Sprintf("%s_ROOT", con.Engine)); sok {
			paths = append(paths, sk)
			env["FAKECHROOT_EXCLUDE_PATH"] = strings.Join(paths, ":")
		}
	}
	return env, nil
}

func (con *Container) bashShell(envmap map[string]string, args ...string) *Error {
	env, err := con.genEnv(envmap)

	if err != nil {
		return err
	}

	if FolderExist(con.RootPath) {
		//here we firstly check if FAKECHROOTKEY is already set, meaning that we are inside fakeroot env as fakeroot does not support nested call
		fakerootkey, fok := os.LookupEnv("FAKEROOTKEY")
		if !fok {
			//we need to start faked-sysv firstly
			faked_sysv := fmt.Sprintf("%s/faked-sysv", con.SysDir)
			foutput, ferr := Command(faked_sysv)
			if ferr != nil {
				LOGGER.WithFields(logrus.Fields{
					"ouput": ferr.Err.Error(),
					"param": faked_sysv,
				}).Error("faked-sysv starts error")
				return ferr
			}
			faked_str := strings.Split(foutput, ":")
			env["FAKEROOTKEY"] = faked_str[0]
			env["FAKEROOTPID"] = faked_str[1]

			//only when we created faked-sysv instance then we need to kill it, otherwise we wait
			defer func() {
				fmt.Sprintf("cleanning up faked-sysv with pid: %s\n", faked_str[1])
				KillProcessByPid(faked_str[1])
			}()
		} else {
			env["FAKEROOTKEY"] = fakerootkey
			env["FAKEROOTPID"] = os.Getenv("FAKEROOTPID")
		}

		LOGGER.WithFields(logrus.Fields{
			"shell":    con.UserShell,
			"env":      env,
			"rootpath": con.RootPath,
			"args":     args,
		}).Debug("bashShell debug, before shellenvpid is called")
		cerr := ShellEnvPid(con.UserShell, env, con.RootPath, args...)
		if cerr != nil {
			return cerr
		}
		return nil
	}
	cerr := ErrNew(ErrNExist, fmt.Sprintf("can't locate container root folder %s", con.RootPath))
	return cerr
}

func (con *Container) createContainer() *Error {
	if con.Id == "" || len(con.Id) == 0 {
		con.Id = RandomString(IDLENGTH)
	}
	con.LogPath = fmt.Sprintf("%s/log", con.ConfigPath)
	con.ElfPatcherPath = con.SysDir
	user, err := Command("whoami")
	if err != nil {
		return err
	}
	con.CreateUser = strings.TrimSuffix(user, "\n")
	_, con.SettingConf, err = LoadConfig(con.SettingPath)
	if err != nil {
		err.AddMsg(fmt.Sprintf("load config from %s encounters error", con.SettingPath))
		return err
	}
	if sh, ok := con.SettingConf["user_shell"]; ok {
		strsh, _ := sh.(string)
		if strings.HasSuffix(strsh, "/") {
			con.UserShell = strsh
		} else {
			shpath, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), strsh, true)
			if err != nil {
				return err
			}
			con.UserShell = shpath
		}
	} else {
		con.UserShell = "/bin/bash"
	}

	con.CurrentUser = "root"

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
										vv1_abs, err := GuessPath(con.RootPath, vv1.(string), true)
										if err != nil {
											continue
										}
										libs = append(libs, vv1_abs)
									}
									if k1, ok1 := k.(string); ok1 {
										k1_abs, err := GuessPath(con.RootPath, k1, true)
										if err != nil {
											continue
										}
										err = con.refreshElf(op, libs, k1_abs)
										if err != nil {
											return err
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
										vv1_abs, err := GuessPath(con.RootPath, vv1.(string), false)
										if err != nil {
											continue
										}
										libs = append(libs, vv1_abs)
									}
								}

								k_abs := AddConPath(con.RootPath, k.(string))
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
	currdir, err := GetConfigDir()
	if err != nil {
		return err
	}
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err = unmarshalObj(rootdir, &sys)

	if err == nil {
		//update or add
		if value, ok := sys.Containers[con.Id]; !ok {
			cmap := make(map[string]string)
			cmap["RootPath"] = con.RootPath
			cmap["SettingPath"] = con.SettingPath
			cmap["ConfigPath"] = con.ConfigPath
			cmap["ContainerID"] = con.Id
			cmap["BaseLayerPath"] = con.BaseLayerPath
			cmap["ContainerName"] = con.ContainerName
			cmap["RPC"] = strconv.Itoa(con.RPCPort)
			cmap["BaseType"] = con.BaseType
			cmap["Image"] = con.ImageBase
			cmap["DataSyncFolder"] = con.DataSyncFolder
			cmap["DataSyncMap"] = con.DataSyncMap
			cmap["BaseType"] = con.BaseType
			cmap["Engine"] = con.Engine
			sys.Containers[con.Id] = cmap
		} else {
			vvalue, vok := value.(map[string]interface{})
			if !vok {
				cerr := ErrNew(ErrType, fmt.Sprintf("appendToSys container type mismatch, actual: %T, should be map[string]string", value))
				return cerr
			}
			vvalue["RootPath"] = con.RootPath
			vvalue["SettingPath"] = con.SettingPath
			vvalue["ConfigPath"] = con.ConfigPath
			vvalue["Image"] = con.ImageBase
			vvalue["BaseType"] = con.BaseType
			sys.Containers[con.Id] = vvalue
		}
		sys.MemcachedPid = fmt.Sprintf("%s/.memcached.pid", sys.RootDir)
		servers := []string{sys.MemcachedPid}
		con.MemcachedServerList = servers
		con.SysDir = rootdir
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

		//set allow_list env
		if ac, ac_ok := con.SettingConf["allow_list"]; ac_ok {
			if aca, aca_ok := ac.([]interface{}); aca_ok {
				for _, ace := range aca {
					if acm, acm_ok := ace.(map[interface{}]interface{}); acm_ok {
						for k, v := range acm {
							k, err := GuessPath(con.RootPath, k.(string), true)
							if err != nil {
								return err
							}
							switch v.(type) {
							case string:
								v_path, v_err := GuessPath(con.RootPath, v.(string), false)
								if v_err != nil {
									LOGGER.WithFields(logrus.Fields{
										"key":   k,
										"value": v.(string),
										"err":   v_err,
										"type":  "string",
									}).Error("allow list parse error")
									continue
								}
								mem_err := mem.MUpdateStrValue(fmt.Sprintf("allow:%s:%s", con.Id, k), v_path)
								if mem_err != nil {
									return mem_err
								}
							case interface{}:
								if acs, acs_ok := v.([]interface{}); acs_ok {
									value := ""
									for _, acl := range acs {
										v_path, v_err := GuessPath(con.RootPath, acl.(string), false)
										if v_err != nil {
											LOGGER.WithFields(logrus.Fields{
												"key":   k,
												"value": acl.(string),
												"err":   v_err,
												"type":  "interface",
											}).Error("allow list parse error")
											continue
										}
										value = fmt.Sprintf("%s;%s", v_path, value)
									}
									mem_err := mem.MUpdateStrValue(fmt.Sprintf("allow:%s:%s", con.Id, k), value)
									if mem_err != nil {
										return mem_err
									}
								}
							default:
								acm_err := ErrNew(ErrType, fmt.Sprintf("allow_list: type is not right, assume: map[interfacer{}]interface{}, real: %v", ace))
								return acm_err
							}
						}
					} else {
						acm_err := ErrNew(ErrType, fmt.Sprintf("allow_list: type is not right, assume: map[string]interface{}, real: %v", ac))
						return acm_err
					}
				}
			} else {
				aca_err := ErrNew(ErrType, fmt.Sprintf("allow_list: type is not right, assume: []interface{}, real: %v", ac))
				return aca_err
			}
		}

		//set deny_list env
		if ac, ac_ok := con.SettingConf["deny_list"]; ac_ok {
			if aca, aca_ok := ac.([]interface{}); aca_ok {
				for _, ace := range aca {
					if acm, acm_ok := ace.(map[interface{}]interface{}); acm_ok {
						for k, v := range acm {
							k, err := GuessPath(con.RootPath, k.(string), true)
							if err != nil {
								return err
							}
							switch v.(type) {
							case string:
								v_path, v_err := GuessPath(con.RootPath, v.(string), false)
								if v_err != nil {
									LOGGER.WithFields(logrus.Fields{
										"key":   k,
										"value": v.(string),
										"err":   v_err,
										"type":  "string",
									}).Error("deny list parse error")
									continue
								}
								mem_err := mem.MUpdateStrValue(fmt.Sprintf("deny:%s:%s", con.Id, k), v_path)
								if mem_err != nil {
									return mem_err
								}
							case interface{}:
								if acs, acs_ok := v.([]interface{}); acs_ok {
									value := ""
									for _, acl := range acs {
										v_path, v_err := GuessPath(con.RootPath, acl.(string), false)
										if v_err != nil {
											LOGGER.WithFields(logrus.Fields{
												"key":   k,
												"value": acl.(string),
												"err":   v_err,
												"type":  "interface",
											}).Error("deny list parse error")
											continue
										}
										value = fmt.Sprintf("%s;%s", v_path, value)
									}
									mem_err := mem.MUpdateStrValue(fmt.Sprintf("deny:%s:%s", con.Id, k), value)
									if mem_err != nil {
										return mem_err
									}
								}
							default:
								acm_err := ErrNew(ErrType, fmt.Sprintf("deny_list: type is not right, assume: map[interfacer{}]interface{}, real: %v", ace))
								return acm_err
							}
						}
					} else {
						acm_err := ErrNew(ErrType, fmt.Sprintf("deny_list: type is not right, assume: map[string]interface{}, real: %v", ac))
						return acm_err
					}
				}
			} else {
				aca_err := ErrNew(ErrType, fmt.Sprintf("deny_list: type is not right, assume: []interface{}, real: %v", ac))
				return aca_err
			}
		}

		//set add_map
		if ac, ac_ok := con.SettingConf["add_map"]; ac_ok {
			if aca, aca_ok := ac.([]interface{}); aca_ok {
				for _, ace := range aca {
					if acm, acm_ok := ace.(map[interface{}]interface{}); acm_ok {
						for k, v := range acm {
							k, err := GuessPath(con.RootPath, k.(string), true)
							if err != nil {
								return err
							}
							switch v.(type) {
							case string:
								v_path, v_err := GuessPath(con.RootPath, v.(string), false)
								if v_err != nil {
									LOGGER.WithFields(logrus.Fields{
										"key":   k,
										"value": v.(string),
										"err":   v_err,
										"type":  "string",
									}).Error("add map parse error")
									continue
								}
								mem_err := mem.MUpdateStrValue(fmt.Sprintf("map:%s:%s", con.Id, k), v_path)
								if mem_err != nil {
									return mem_err
								}
							case interface{}:
								if acs, acs_ok := v.([]interface{}); acs_ok {
									value := ""
									for _, acl := range acs {
										v_path, v_err := GuessPath(con.RootPath, acl.(string), false)
										if v_err != nil {
											LOGGER.WithFields(logrus.Fields{
												"key":   k,
												"value": acl.(string),
												"err":   v_err,
												"type":  "interface",
											}).Error("add map parse error")
											continue
										}
										value = fmt.Sprintf("%s;%s", v_path, value)
									}
									mem_err := mem.MUpdateStrValue(fmt.Sprintf("map:%s:%s", con.Id, k), value)
									if mem_err != nil {
										return mem_err
									}
								}
							default:
								acm_err := ErrNew(ErrType, fmt.Sprintf("add_map: type is not right, assume: map[interfacer{}]interface{}, real: %v", ace))
								return acm_err
							}
						}
					} else {
						acm_err := ErrNew(ErrType, fmt.Sprintf("add_map: type is not right, assume: map[string]interface{}, real: %v", ac))
						return acm_err
					}
				}
			} else {
				aca_err := ErrNew(ErrType, fmt.Sprintf("add_map: type is not right, assume: []interface{}, real: %v", ac))
				return aca_err
			}
		}

		return nil
	} else {
		mem_err := ErrNew(err, "memcache server init error")
		return mem_err
	}
	return err
}

func (con *Container) startRPCService(port int) *Error {
	con.RPCMap = make(map[int]string)
	conn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		cerr := ErrNew(err, "start rpc service encounters error")
		return cerr
	}
	env := make(map[string]string)
	env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so", con.SysDir)
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

//this function will generate necessary info and save it to file. This function should work properly in order to create a container
/**
name: image name and tag
container_name: optional container name
volume_map: a volume map for container
command: command to run inside container
**/
func generateContainer(name, container_name, volume_map, engine string) (*map[string]interface{}, *Error) {
	currdir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	rootdir := fmt.Sprintf("%s/.lpmxdata", currdir)
	var doc Image
	err = unmarshalObj(rootdir, &doc)
	if err == nil {
		if val, ok := doc.Images[name]; ok {
			if vval, vok := val.(map[string]interface{}); vok {
				//base is LPMX/.lpmxdata/.base
				base, _ := vval["base"].(string)
				workspace, _ := vval["workspace"].(string)
				config, _ := vval["config"].(string)
				layers, _ := vval["layer_order"].(string)
				imagetype, _ := vval["imagetype"].(string)
				//randomly generate id
				id := RandomString(IDLENGTH)
				//rootfolder is the folder containing ld.so.path, rw and other layers symlinks
				rootfolder := fmt.Sprintf("%s/%s", workspace, id)
				if !FolderExist(rootfolder) {
					_, err := MakeDir(rootfolder)
					if err != nil {
						return nil, err
					}
				}

				//create symlink folder for different layers
				var keys []string
				for _, k := range strings.Split(layers, ":") {
					k = path.Base(k)
					src_path := fmt.Sprintf("%s/%s", base, k)
					target_path := fmt.Sprintf("%s/%s", rootfolder, k)
					err := os.Symlink(src_path, target_path)
					if err != nil {
						cerr := ErrNew(err, fmt.Sprintf("can't create symlink from path: %s to %s", src_path, target_path))
						return nil, cerr
					}
					keys = append(keys, k)
				}
				keys = append(keys, "rw")
				configmap := make(map[string]interface{})
				configmap["dir"] = fmt.Sprintf("%s/rw", rootfolder)
				if !FolderExist(configmap["dir"].(string)) {
					_, err := MakeDir(configmap["dir"].(string))
					if err != nil {
						return nil, err
					}
				}
				configmap["parent_dir"] = rootfolder

				configmap["config"] = config
				configmap["passive"] = false
				configmap["id"] = id
				configmap["image"] = name
				configmap["imagetype"] = imagetype
				LOGGER.WithFields(logrus.Fields{
					"keys":   keys,
					"layers": layers,
				}).Debug("layers sha256 list")
				reverse_keys := ReverseStrArray(keys)
				configmap["layers"] = strings.Join(reverse_keys, ":")
				configmap["baselayerpath"] = base
				configmap["container_name"] = container_name
				if _, fok := FindStrinArray(engine, ENGINE_TYPE); fok {
					//set engine type
					configmap["engine"] = strings.ToUpper(engine)
					//enable engine
					configmap["enable_engine"] = "True"
				}

				//dealing with sync folder/volume problem
				//add default sync folder firstly if not exists
				var sync_folder []string
				default_sync_folder := fmt.Sprintf("%s/sync/%s", currdir, id)
				if !FolderExist(default_sync_folder) {
					oerr := os.MkdirAll(default_sync_folder, os.FileMode(FOLDER_MODE))
					if oerr != nil {
						cerr := ErrNew(oerr, fmt.Sprintf("could not mkdir %s", default_sync_folder))
						return nil, cerr
					}
				}
				if !strings.Contains(volume_map, default_sync_folder) {
					volume_map = fmt.Sprintf("%s:/lpmx;%s", default_sync_folder, volume_map)
				}
				//add user defined ones
				for _, volume := range strings.Split(volume_map, ";") {
					if len(volume) > 0 {
						v := strings.Split(volume, ":")
						if !FolderExist(v[0]) {
							continue
						} else {
							v_abs := fmt.Sprintf("%s/rw%s", rootfolder, v[1])
							if FolderExist(v_abs) {
								continue
							} else {
								serr := os.Symlink(v[0], v_abs)
								if serr != nil {
									cerr := ErrNew(serr, fmt.Sprintf("could not symlink, oldpath: %s, newpath: %s", v[0], v_abs))
									return nil, cerr
								}
								sync_folder = append(sync_folder, v[0])
							}
						}
					}
				}
				configmap["sync_ori_folder"] = volume_map
				configmap["sync_folder"] = strings.Join(sync_folder, ":")
				if len(sync_folder) == 1 {
					configmap["sync_folder"] = strings.TrimSuffix(configmap["sync_folder"].(string), ":")
				}

				//patch ld.so
				//update on 20191223 we downloaded patch.tar.gz from github and we need to patch ld.so included inside this tar ball rather than using the one inside container
				ld_new_path := fmt.Sprintf("%s/ld.so.patch", rootfolder)
				LOGGER.WithFields(logrus.Fields{
					"ld_patched_path": ld_new_path,
				}).Debug("layers sha256 list")
				if !FileExist(ld_new_path) {
					//update on 20200120 we do not need to rebuild ld.so again. As we found that __libc_start_main can be LD_PRELOAD and trapped before main function is called.
					//will comment the following part of code

					/**
					ld_orig_path := fmt.Sprintf("%s/patch/ld.so", filepath.Dir(filepath.Dir(rootfolder)))
					LOGGER.WithFields(logrus.Fields{
						"ld_path": ld_orig_path,
					}).Debug("DockerCreate prepares patching target ld.so")
					_, err := os.Stat(ld_orig_path)
					if err == nil {
						perr := Patchldso(ld_orig_path, ld_new_path)
						if perr != nil {
							return perr
						}
						configmap["elf_loader"] = ld_new_path
					} else {
						cerr := ErrNew(err, fmt.Sprintf("could not patch target ld.so: %s", ld_orig_path))
						return cerr
					}
					**/
					///**
					for _, v := range LD {
						for _, l := range strings.Split(configmap["layers"].(string), ":") {
							ld_orig_path := fmt.Sprintf("%s/%s%s", configmap["baselayerpath"].(string), l, v)

							LOGGER.WithFields(logrus.Fields{
								"ld_path": ld_orig_path,
							}).Debug("layers sha256 list")
							if _, err := os.Stat(ld_orig_path); err == nil {
								err := Patchldso(ld_orig_path, ld_new_path)
								if err != nil {
									return nil, err
								}
								configmap["elf_loader"] = ld_new_path
								break
							}
						}
						if _, ok := configmap["elf_loader"]; ok {
							break
						}
					}
					//**/
				} else {
					configmap["elf_loader"] = ld_new_path
				}

				//add current user to /etc/passwd user gid to /etc/group
				user, err := user.Current()
				if err != nil {
					cerr := ErrNew(err, "can't get current user info")
					return nil, cerr
				}

				LOGGER.WithFields(logrus.Fields{
					"configmap": configmap,
				}).Debug("configmap info debugging before copy and create /etc/passwd and /etc/group")

				uname := user.Username
				uid := user.Uid
				gid := user.Gid
				passwd_patch := false
				group_patch := false
				for _, l := range strings.Split(configmap["layers"].(string), ":") {
					passwd_path := fmt.Sprintf("%s/%s/etc/passwd", configmap["baselayerpath"].(string), l)
					if _, err := os.Stat(passwd_path); err == nil {
						new_passwd_path := fmt.Sprintf("%s/etc", configmap["dir"].(string))
						os.MkdirAll(new_passwd_path, os.FileMode(FOLDER_MODE))
						ret, c_err := CopyFile(passwd_path, fmt.Sprintf("%s/passwd", new_passwd_path))
						if ret && c_err == nil {
							f, err := os.OpenFile(fmt.Sprintf("%s/passwd", new_passwd_path), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/passwd", new_passwd_path))
								return nil, cerr
							}
							defer f.Close()
							_, err = f.WriteString(fmt.Sprintf("%s:x:%s:%s:%s:/home/%s:/bin/bash\n", uname, uid, uid, uname, uname))
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/passwd", new_passwd_path))
								return nil, cerr
							}

							passwd_patch = true
							break
						} else {
							return nil, c_err
						}
					}
				}

				for _, l := range strings.Split(configmap["layers"].(string), ":") {
					group_path := fmt.Sprintf("%s/%s/etc/group", configmap["baselayerpath"].(string), l)
					if _, err := os.Stat(group_path); err == nil {
						new_group_path := fmt.Sprintf("%s/etc", configmap["dir"].(string))
						os.MkdirAll(new_group_path, os.FileMode(FOLDER_MODE))
						ret, c_err := CopyFile(group_path, fmt.Sprintf("%s/group", new_group_path))
						if ret && c_err == nil {
							f, err := os.OpenFile(fmt.Sprintf("%s/group", new_group_path), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/group", new_group_path))
								return nil, cerr
							}
							defer f.Close()
							_, err = f.WriteString(fmt.Sprintf("%s:x:%s\n", uname, gid))
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/group", new_group_path))
								return nil, cerr
							}

							host_f, err := os.OpenFile("/etc/group", os.O_RDONLY, 0400)
							if err == nil {
								scanner := bufio.NewScanner(host_f)
								for scanner.Scan() {
									content := scanner.Text()
									if strings.HasSuffix(content, fmt.Sprintf(":%s", uname)) {
										f.WriteString(fmt.Sprintf("%s\n", content))
									}
								}
								host_f.Close()
							}

							group_patch = true
							break
						} else {
							return nil, c_err
						}
					}
				}

				if !passwd_patch || !group_patch {
					cerr := ErrNew(ErrNExist, "could not find /etc/passwd or /etc/group to patch")
					return nil, cerr
				}

				//create tmp folder and create whiteout file for tmp
				os.MkdirAll(fmt.Sprintf("%s/tmp", configmap["dir"].(string)), os.FileMode(FOLDER_MODE))
				f, _ := os.Create(fmt.Sprintf("%s/.wh.tmp", configmap["dir"].(string)))
				f.Close()

				//run container
				return &configmap, nil
			}
		} //if image exists inside doc data structure
		cerr := ErrNew(ErrNExist, fmt.Sprintf("image %s doesn't exist", name))
		return nil, cerr
	}
	if err.Err == ErrNExist {
		err.AddMsg(fmt.Sprintf("image %s does not exist, you may need to download it firstly", name))
	}
	return nil, err

}

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

func walkfsandcopy(src, dest string, excludes []string) *Error {
	if !FolderExist(src) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("source folder: %s does not exist", src))
		return cerr
	}

	if !FolderExist(dest) {
		err := os.MkdirAll(dest, os.FileMode(FOLDER_MODE))
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not create dest dir: %s", dest))
			return cerr
		}
	}

	err := filepath.Walk(src, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() && f.Name() == ".lpmx" {
			return filepath.SkipDir
		}
		file_type, ferr := FileType(path)
		if ferr != nil {
			return ferr.Err
		}

		filename := filepath.Base(path)
		bskip := false
		for _, exclude := range excludes {
			if filename == exclude {
				bskip = true
				break
			}
		}

		if !bskip {
			dest_file := fmt.Sprintf("%s/%s", dest, filename)
			//if normal regular file, directly copy it to target folder
			if file_type == TYPE_REGULAR {
				_, cerr := CopyFile(path, dest_file)
				if cerr != nil {
					return cerr.Err
				}
			}
			//if normal symlink file, create symlink file
			if file_type == TYPE_SYMLINK {
				link, lerr := os.Readlink(path)
				if lerr != nil {
					cerr := ErrNew(lerr, fmt.Sprintf("could not successfully resolve symlink: %s", path))
					return cerr.Err
				}
				oerr := os.Symlink(link, dest_file)
				if oerr != nil {
					cerr := ErrNew(oerr, fmt.Sprintf("could not create symlink %s -> %s", dest_file, link))
					return cerr.Err
				}
			}
		}

		return nil
	})

	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("filepath walk: %s encounters error", src))
		return cerr
	}
	return nil
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
		return nil, cerr
	}
	return fileList, nil
}

func unmarshalObj(rootdir string, inf interface{}) *Error {
	info := fmt.Sprintf("%s/.info", rootdir)
	if FileExist(info) {
		data, err := ReadFromFile(info)
		if err != nil {
			return err
		}
		switch inf.(type) {
		case *Sys:
			err = StructUnmarshal(data, inf.(*Sys))
		case *Container:
			err = StructUnmarshal(data, inf.(*Container))
		case *Image:
			err = StructUnmarshal(data, inf.(*Image))
		case *ImageInfo:
			err = StructUnmarshal(data, inf.(*ImageInfo))
		default:
			cerr := ErrNew(ErrMismatch, "interface type mismatched, should be *Sys, *Docker or *Container")
			return cerr
		}
		if err != nil {
			return err
		}
	} else {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.info doesn't exist", rootdir))
		return cerr
	}
	return nil
}

func setPrivilege(id string, tp string, name string, value string, server string, allow bool) *Error {
	mem, err := MInitServers(server)
	if err != nil {
		return err
	}

	if allow {
		if tp == ELFOP[0] {
			err := mem.MUpdateStrValue(fmt.Sprintf("allow:%s:%s", id, name), value)
			if err != nil {
				return err
			}
		}
		if tp == ELFOP[1] {
			err := mem.MDeleteByKey(fmt.Sprintf("allow:%s:%s", id, name))
			if err != nil {
				return err
			}
		}
	} else {
		if tp == ELFOP[2] {
			err := mem.MUpdateStrValue(fmt.Sprintf("deny:%s:%s", id, name), value)
			if err != nil {
				return err
			}
		}
		if tp == ELFOP[3] {
			err := mem.MDeleteByKey(fmt.Sprintf("deny:%s:%s", id, name))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getPrivilege(id string, name string, server string, allow bool) (string, *Error) {
	mem, err := MInitServers(server)
	if err != nil {
		return "", err
	}

	var str string
	if allow {
		str, err = mem.MGetStrValue(fmt.Sprintf("allow:%s:%s", id, name))
	} else {
		str, err = mem.MGetStrValue(fmt.Sprintf("deny:%s:%s", id, name))
	}
	if err != nil {
		return "", err
	}
	return str, nil
}

func setExec(id, tp, name, value, server string, mode bool) *Error {
	if !mode {
		//read config location
		currdir, err := GetConfigDir()
		if err != nil {
			return err
		}
		var sys Sys
		rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
		err = unmarshalObj(rootdir, &sys)
		if err != nil {
			return err
		}
		if v, ok := sys.Containers[id]; ok {
			if vval, vok := v.(map[string]interface{}); vok {
				configpath := vval["ConfigPath"].(string)
				filepath := fmt.Sprintf("%s/.execmap", configpath)
				fc, ferr := FInitServer(filepath)
				if ferr != nil {
					return ferr
				}

				if tp == ELFOP[6] {
					ferr = fc.FSetValue(name, value)
					if ferr != nil {
						return ferr
					}
				}

				if tp == ELFOP[7] {
					ferr = fc.FDeleteByKey(name)
					if ferr != nil {
						return ferr
					}
				}

			}
		}
	} else {
		mem, err := MInitServers(server)
		if err != nil {
			return err
		}

		if tp == ELFOP[6] {
			err := mem.MUpdateStrValue(fmt.Sprintf("exec:%s:%s", id, name), value)
			if err != nil {
				return err
			}
		}

		if tp == ELFOP[7] {
			err := mem.MDeleteByKey(fmt.Sprintf("exec:%s:%s", id, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getExec(id, name, server string, mode bool) (string, *Error) {
	if mode {
		//read config location
		currdir, err := GetConfigDir()
		if err != nil {
			return "", err
		}
		var sys Sys
		rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
		err = unmarshalObj(rootdir, &sys)
		if err != nil {
			return "", err
		}
		if v, ok := sys.Containers[id]; ok {
			if vval, vok := v.(map[string]interface{}); vok {
				configpath := vval["ConfigPath"].(string)
				filepath := fmt.Sprintf("%s/.execmap", configpath)
				fc, ferr := FInitServer(filepath)
				if ferr != nil {
					return "", ferr
				}

				str, serr := fc.FGetStrValue(name)
				if serr != nil {
					return "", serr
				}
				return str, nil
			}
		}
	} else {
		mem, err := MInitServers(server)
		if err != nil {
			return "", err
		}

		str, err := mem.MGetStrValue(fmt.Sprintf("exec:%s:%s", id, name))
		if err != nil {
			return "", err
		}
		return str, nil
	}

	return "", nil
}

func setMap(id string, tp string, name string, value string, server string, mode bool) *Error {
	if !mode {
		//read config location
		currdir, err := GetConfigDir()
		if err != nil {
			return err
		}
		var sys Sys
		rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
		err = unmarshalObj(rootdir, &sys)
		if err != nil {
			return err
		}
		if v, ok := sys.Containers[id]; ok {
			if vval, vok := v.(map[string]interface{}); vok {
				configpath := vval["ConfigPath"].(string)
				filepath := fmt.Sprintf("%s/.mapmap", configpath)
				fc, ferr := FInitServer(filepath)
				if ferr != nil {
					return ferr
				}

				if tp == ELFOP[4] {
					ferr = fc.FSetValue(name, value)
					if ferr != nil {
						return ferr
					}
				}

				if tp == ELFOP[5] {
					ferr = fc.FDeleteByKey(name)
					if ferr != nil {
						return ferr
					}
				}

			}
		}
	} else {
		mem, err := MInitServers(server)
		if err != nil {
			return err
		}

		if tp == ELFOP[4] {
			err := mem.MUpdateStrValue(fmt.Sprintf("map:%s:%s", id, name), value)
			if err != nil {
				return err
			}
		}

		if tp == ELFOP[5] {
			err := mem.MDeleteByKey(fmt.Sprintf("map:%s:%s", id, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getMap(id string, name string, server string, mode bool) (string, *Error) {
	if mode {
		//read config location
		currdir, err := GetConfigDir()
		if err != nil {
			return "", err
		}
		var sys Sys
		rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
		err = unmarshalObj(rootdir, &sys)
		if err != nil {
			return "", err
		}
		if v, ok := sys.Containers[id]; ok {
			if vval, vok := v.(map[string]interface{}); vok {
				configpath := vval["ConfigPath"].(string)
				filepath := fmt.Sprintf("%s/.mapmap", configpath)
				fc, ferr := FInitServer(filepath)
				if ferr != nil {
					return "", ferr
				}

				str, serr := fc.FGetStrValue(name)
				if serr != nil {
					return "", serr
				}
				return str, nil
			}
		}
	} else {
		mem, err := MInitServers(server)
		if err != nil {
			return "", err
		}

		str, err := mem.MGetStrValue(fmt.Sprintf("map:%s:%s", id, name))
		if err != nil {
			return "", err
		}
		return str, nil
	}
	return "", nil
}

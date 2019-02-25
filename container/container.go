package container

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/jasonyangshadow/lpmx/docker"
	. "github.com/jasonyangshadow/lpmx/elf"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/log"
	. "github.com/jasonyangshadow/lpmx/memcache"
	. "github.com/jasonyangshadow/lpmx/msgpack"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/pid"
	. "github.com/jasonyangshadow/lpmx/rpc"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"github.com/sirupsen/logrus"
)

const (
	IDLENGTH = 10
)

var (
	ELFOP                   = []string{"add_allow_priv", "remove_allow_priv", "add_deny_priv", "remove_deny_priv", "add_map", "remove_map"}
	LD                      = []string{"/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2", "/lib/ld.so", "/lib64/ld-linux-x86-64.so.2", "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.1", "/lib64/ld-linux-x86-64.so.1", "/lib/ld-linux.so.2", "/lib/ld-linux.so.1"}
	LD_LIBRARY_PATH_DEFAULT = []string{"lib", "lib/x86_64-linux-gnu", "usr/lib/x86_64-linux-gnu", "usr/lib", "usr/local/lib"}
	FOLDER_MODE             = 0755
	CACHE_FOLDER            = []string{"/var/cache/apt/archives"}
	UNSTALL_FOLDER          = []string{".lpmxsys", "sync", "bin", ".docker", "package"}
)

//located inside $/.lpmxsys/.info
type Sys struct {
	RootDir      string // the abs path of folder .lpmxsys
	BinaryDir    string // the folder cotnainers .lpmxsys and binaries
	Containers   map[string]interface{}
	LogPath      string
	MemcachedPid string
}

//located inside $/.docker/image/tag/workspace/.lpmx/.info
type Container struct {
	Id                  string
	RootPath            string
	ConfigPath          string
	ImageBase           string
	DockerBase          bool
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
}

type RPC struct {
	Env map[string]string
	Dir string
	Con *Container
}

//used for storing all docker images, located inside $/.docker/.info
type Docker struct {
	RootDir string
	Images  map[string]interface{}
}

//used for offline docker image installation, located inside $/.docker/image/tag/.info
type DockerInfo struct {
	Name      string
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
	currdir, _ := GetCurrDir()
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
	sys.BinaryDir = currdir
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
			fmt.Println("Downloading dependency.tar.gz from github")
			err = DownloadFilefromGithub(dist, release, "dependency.tar.gz", SETTING_URL, sys.RootDir)
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
	currdir, _ := GetCurrDir()
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

func List() *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)
	if err == nil {
		fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", "ContainerID", "ContainerName", "Status", "PID", "RPC", "DockerBase", "Image"))
		for k, v := range sys.Containers {
			if cmap, ok := v.(map[string]interface{}); ok {
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
							fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "RUNNING", strconv.Itoa(pid), cmap["RPC"].(string), cmap["DockerBase"].(string), cmap["Image"].(string)))
						} else {
							fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "STOPPED", "NA", cmap["RPC"].(string), cmap["DockerBase"].(string), cmap["Image"].(string)))
						}
					}
				} else {
					if pid != -1 {
						fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "RUNNING", strconv.Itoa(pid), "NA", cmap["DockerBase"].(string), cmap["Image"].(string)))
					} else {
						fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%15s%15s", k, cmap["ContainerName"].(string), "STOPPED", "NA", "NA", cmap["DockerBase"].(string), cmap["Image"].(string)))
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

func Resume(id string, args ...string) *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)
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
					if con.DockerBase {
						configmap["docker"] = true
						configmap["layers"] = con.Layers
						configmap["id"] = con.Id
						configmap["image"] = con.ImageBase
						configmap["baselayerpath"] = con.BaseLayerPath
						configmap["elf_loader"] = con.PatchedELFLoader
						configmap["parent_dir"] = filepath.Dir(con.RootPath)
						configmap["sync_folder"] = con.DataSyncFolder
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
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)

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
					//check if container is based on docker
					docker, _ := val["DockerBase"].(string)
					if dockerb, _ := strconv.ParseBool(docker); dockerb {
						rootdir, _ := val["RootPath"].(string)
						rootdir = path.Dir(rootdir)
						RemoveAll(rootdir)
					} else {
						cdir := fmt.Sprintf("%s/.lpmx", val["RootPath"])
						RemoveAll(cdir)
					}
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
	//dir is rw folder of container
	dir, _ := (*configmap)["dir"].(string)
	config, _ := (*configmap)["config"].(string)
	passive, _ := (*configmap)["passive"].(bool)

	//parent dir is the folder containing rw, base layers and ld.so.patch
	parent_dir, _ := (*configmap)["parent_dir"].(string)
	rootdir := fmt.Sprintf("%s/.lpmx", parent_dir)

	var con Container
	//used for distinguishing the docker based and non-docker based container
	if dockerbase, dok := (*configmap)["docker"]; dok {
		dbase, _ := dockerbase.(bool)
		if dbase {
			con.Id = (*configmap)["id"].(string)
			con.DockerBase = true
			con.ImageBase = (*configmap)["image"].(string)
			con.Layers = (*configmap)["layers"].(string)
			con.BaseLayerPath = (*configmap)["baselayerpath"].(string)
			con.PatchedELFLoader = (*configmap)["elf_loader"].(string)
			con.DataSyncFolder = (*configmap)["sync_folder"].(string)
		}
	} else {
		con.DockerBase = false
	}
	con.RootPath = dir
	con.ConfigPath = rootdir
	con.SettingPath = config
	if (*configmap)["container_name"] == nil {
		(*configmap)["container_name"] = ""
	}
	con.ContainerName = (*configmap)["container_name"].(string)

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
		err = con.bashShell(args...)
		if err != nil {
			err.AddMsg("starting bash shell encounters error")
			return err
		}
	}

	return nil
}

func Get(id string, name string) *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)

	if err == nil {
		if _, ok := sys.Containers[id]; ok {
			fmt.Println(fmt.Sprintf("|%-s|%-30s|%-20s|%-20s|%-10s|", "ContainerID", "PROGRAM", "ALLOW_PRIVILEGES", "DENY_PRIVILEGES", "REMAP"))
			a_val, _ := getPrivilege(id, name, sys.MemcachedPid, true)
			d_val, _ := getPrivilege(id, name, sys.MemcachedPid, false)
			m_val, _ := getMap(id, name, sys.MemcachedPid)
			fmt.Println(fmt.Sprintf("|%-s|%-30s|%-20s|%-20s|%-10s|", id, name, a_val, d_val, m_val))
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

func Set(id string, tp string, name string, value string) *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if _, vok := v.(map[string]interface{}); vok {
				tp = strings.ToLower(strings.TrimSpace(tp))
				switch tp {
				case ELFOP[4], ELFOP[5]:
					{
						err := setMap(id, tp, name, value, sys.MemcachedPid)
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
					err_new := ErrNew(ErrType, "tp should be one of 'add_allow_priv','remove_allow_priv','add_deny_priv','remove_deny_priv','add_map','remove_map'}")
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
	currdir, _ := GetCurrDir()
	packagedir := fmt.Sprintf("%s/package", currdir)
	if !FolderExist(packagedir) {
		MakeDir(packagedir)
	}
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := unmarshalObj(rootdir, &doc)
	if err != nil {
		return err
	}
	if dvalue, dok := doc.Images[name]; dok {
		if dmap, dmok := dvalue.(map[string]interface{}); dmok {
			var filelist []string
			filelist = append(filelist, dmap["config"].(string))
			layers := strings.Split(dmap["layer_order"].(string), ":")
			layer_base := dmap["image"].(string)
			for _, layer := range layers {
				layer = path.Base(layer)
				filelist = append(filelist, fmt.Sprintf("%s/%s", layer_base, layer))
			}
			filelist = append(filelist, fmt.Sprintf("%s/.info", dmap["rootdir"].(string)))
			cerr := TarFiles(filelist, packagedir, name)
			if cerr != nil {
				return cerr
			}
		} else {
			cerr := ErrNew(ErrMismatch, "type mismatched")
			return cerr
		}
	} else {
		cerr := DockerDownload(name, user, pass)
		if cerr != nil && cerr.Err != ErrExist {
			return cerr
		}
	}
	return nil
}

func DockerAdd(file string) *Error {
	if !FileExist(file) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does not exist", file))
		return cerr
	}
	currdir, _ := GetCurrDir()
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := unmarshalObj(rootdir, &doc)
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
	dir, derr := ioutil.TempDir("", "lpmx")
	if derr != nil {
		cerr := ErrNew(derr, "could not create temp dir")
		return cerr
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
	var docinfo DockerInfo
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
		mdata["image"] = fmt.Sprintf("%s/image", mdata["rootdir"])
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
			err := os.Rename(lay_path, lay_new_path)
			if err != nil {
				cerr := ErrNew(err, fmt.Sprintf("could not move file %s to %s", lay_path, lay_new_path))
				return cerr
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
		base := fmt.Sprintf("%s/base", mdata["rootdir"])
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
			}

			err := Untar(tar_path, layerfolder)
			if err != nil {
				return err
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

		ddata, _ := StructMarshal(doc)
		cerr := WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
		if cerr != nil {
			return cerr
		}
		return nil
	}
}

func DockerCommit(id, newname, newtag string) *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	//first check whether the container is running
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)
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
					rootdir := fmt.Sprintf("%s/.docker", currdir)
					var doc Docker
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
					//moving /etc/group /etc/passwd /proc /tmp folder to temp folder
					cache_temp_dir, _ := ioutil.TempDir("", "lpmx")
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
					if FolderExist(fmt.Sprintf("%s/proc", con.RootPath)) {
						cerr := Rename(fmt.Sprintf("%s/proc", con.RootPath), fmt.Sprintf("%s/proc", cache_temp_dir))
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
					if FolderExist(fmt.Sprintf("%s/lpmx", con.RootPath)) {
						RemoveAll(fmt.Sprintf("%s/lpmx", con.RootPath))
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
					temp_dir, _ := ioutil.TempDir("", "lpmx")
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
					//image dir is LPMX/.docker/.image
					//moving layer tarball to image folder
					fmt.Println("renaming rw layer...")
					image_dir := fmt.Sprintf("%s/.image", filepath.Dir(con.BaseLayerPath))
					src_tar_path := rw_tar_path
					target_tar_path := fmt.Sprintf("%s/%s", image_dir, shasum)
					rerr := os.Rename(src_tar_path, target_tar_path)
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not rename(move): %s to %s", src_tar_path, target_tar_path))
						return cerr
					}

					//moving rw layer to base folder
					rerr = os.Rename(con.RootPath, fmt.Sprintf("%s/%s", con.BaseLayerPath, shasum))
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not rename(move): %s to %s", con.RootPath, fmt.Sprintf("%s/%s", con.BaseLayerPath, shasum)))
						return cerr
					}

					//create new symlink
					new_symlink_path := fmt.Sprintf("%s/%s", filepath.Dir(con.RootPath), shasum)
					old_symlink_path := fmt.Sprintf("%s/%s", con.BaseLayerPath, shasum)
					rerr = os.Symlink(old_symlink_path, new_symlink_path)
					if rerr != nil {
						cerr := ErrNew(rerr, fmt.Sprintf("could not symlink: %s to %s", old_symlink_path, new_symlink_path))
						return cerr
					}

					//moving workspace and copyting setting.yml to new place
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
					derr = os.Symlink(con.DataSyncFolder, fmt.Sprintf("%s/lpmx", con.RootPath))
					if derr != nil {
						cerr := ErrNew(derr, fmt.Sprintf("could not symlink, oldpath: %s, newpath: %s", con.DataSyncFolder, fmt.Sprintf("%s/lpmx", con.RootPath)))
						return cerr
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
					//create rw/proc/self/cwd to fake cwd
					proc_self_path := fmt.Sprintf("%s/proc/self", con.RootPath)
					os.MkdirAll(proc_self_path, os.FileMode(FOLDER_MODE))
					os.Symlink("/", fmt.Sprintf("%s/cwd", proc_self_path))
					os.Symlink("/", fmt.Sprintf("%s/exe", proc_self_path))
					//create new tmp
					os.MkdirAll(fmt.Sprintf("%s/tmp", con.RootPath), os.FileMode(FOLDER_MODE))
					f, _ := os.Create(fmt.Sprintf("%s/.wh.tmp", con.RootPath))
					f.Close()

					//step 4: modify container info
					new_layers := []string{"rw", shasum}
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

					//located inside $/.docker
					//start updating image info
					fmt.Println("updating image info...")
					mdata := make(map[string]interface{})
					mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", docker_path, newname, newtag)
					mdata["config"] = con.SettingPath
					mdata["image"] = fmt.Sprintf("%s/.image", docker_path)
					//get old layer map
					image_map := doc.Images[old_imagebase]
					if image_map != nil {
						if map_interface, map_ok := image_map.(map[string]interface{}); map_ok {
							if old_map, old_ok := (map_interface["layer"]).(map[string]interface{}); old_ok {
								size, serr := GetFileSize(target_tar_path)
								if serr != nil {
									return serr
								}
								old_map[fmt.Sprintf("%s/%s", mdata["image"].(string), shasum)] = size
								mdata["layer"] = old_map
								mdata["layer_order"] = fmt.Sprintf("%s:%s", map_interface["layer_order"].(string), fmt.Sprintf("%s/%s", mdata["image"].(string), shasum))
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

					doc.Images[fmt.Sprintf("%s:%s", newname, newtag)] = mdata
					mddata, _ := StructMarshal(doc)
					LOGGER.WithFields(logrus.Fields{
						"doc": doc,
					}).Debug("DockerCommit, update image info")
					cerr = WriteToFile(mddata, fmt.Sprintf("%s/.info", doc.RootDir))
					if cerr != nil {
						return cerr
					}

					//start adding docinfo
					var docinfo DockerInfo
					docinfo.Name = fmt.Sprintf("%s:%s", newname, newtag)
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
						"docinfo": docinfo,
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

func DockerDownload(name string, user string, pass string) *Error {
	currdir, _ := GetCurrDir()
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := unmarshalObj(rootdir, &doc)
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

		//add docker info file(.info)
		if !FolderExist(mdata["rootdir"].(string)) {
			merr := os.MkdirAll(mdata["rootdir"].(string), os.FileMode(FOLDER_MODE))
			if merr != nil {
				cerr := ErrNew(merr, fmt.Sprintf("could not mkdir %s", mdata["rootdir"].(string)))
				return cerr
			}
		}
		var docinfo DockerInfo
		docinfo.Name = name
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
		}).Debug("DockerDownload debug, docinfo debug")

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
		err = DownloadFilefromGithub(tname, ttag, "setting.yml", SETTING_URL, rdir)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download setting from github failure and could not rollback to default one")
			return err
		}

		//add map to this image
		doc.Images[name] = mdata

		ddata, _ := StructMarshal(doc)
		err = WriteToFile(ddata, fmt.Sprintf("%s/.info", doc.RootDir))
		if err != nil {
			return err
		}
		return nil
	}
}

func DockerPush(user string, pass string, name string, tag string, id string) *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	//first check whether the container is running
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)
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

func DockerList() *Error {
	currdir, _ := GetCurrDir()
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := unmarshalObj(rootdir, &doc)
	fmt.Println(fmt.Sprintf("%s", "Name"))
	if err == nil {
		for k, _ := range doc.Images {
			fmt.Println(fmt.Sprintf("%s", k))
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
	currdir, _ := GetCurrDir()
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := unmarshalObj(rootdir, &doc)
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

func DockerCreate(name string, container_name string) *Error {
	currdir, _ := GetCurrDir()
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := unmarshalObj(rootdir, &doc)
	if err == nil {
		if val, ok := doc.Images[name]; ok {
			if vval, vok := val.(map[string]interface{}); vok {
				//base is LPMX/.docker/.base
				base, _ := vval["base"].(string)
				workspace, _ := vval["workspace"].(string)
				config, _ := vval["config"].(string)
				layers, _ := vval["layer_order"].(string)
				//randomly generate id
				id := RandomString(IDLENGTH)
				//rootfolder is the folder containing ld.so.path, rw and other layers symlinks
				rootfolder := fmt.Sprintf("%s/%s", workspace, id)
				if !FolderExist(rootfolder) {
					_, err := MakeDir(rootfolder)
					if err != nil {
						return err
					}
				}

				//create symlink folder
				var keys []string
				for _, k := range strings.Split(layers, ":") {
					k = path.Base(k)
					src_path := fmt.Sprintf("%s/%s", base, k)
					target_path := fmt.Sprintf("%s/%s", rootfolder, k)
					err := os.Symlink(src_path, target_path)
					if err != nil {
						cerr := ErrNew(err, fmt.Sprintf("can't create symlink from path: %s to %s", src_path, target_path))
						return cerr
					}
					keys = append(keys, k)
				}
				keys = append(keys, "rw")
				configmap := make(map[string]interface{})
				configmap["dir"] = fmt.Sprintf("%s/rw", rootfolder)
				if !FolderExist(configmap["dir"].(string)) {
					_, err := MakeDir(configmap["dir"].(string))
					if err != nil {
						return err
					}
				}
				configmap["parent_dir"] = rootfolder

				configmap["config"] = config
				configmap["passive"] = false
				configmap["id"] = id
				configmap["image"] = name
				configmap["docker"] = true
				LOGGER.WithFields(logrus.Fields{
					"keys":   keys,
					"layers": layers,
				}).Debug("layers sha256 list")
				reverse_keys := ReverseStrArray(keys)
				configmap["layers"] = strings.Join(reverse_keys, ":")
				configmap["baselayerpath"] = base
				configmap["container_name"] = container_name
				configmap["sync_folder"] = fmt.Sprintf("%s/sync/%s", currdir, id)
				if !FolderExist(configmap["sync_folder"].(string)) {
					oerr := os.MkdirAll(configmap["sync_folder"].(string), os.FileMode(FOLDER_MODE))
					if oerr != nil {
						cerr := ErrNew(oerr, fmt.Sprintf("could not mkdir %s", configmap["sync_folder"].(string)))
						return cerr
					}
				}
				//create symlink inside rw folder to host
				serr := os.Symlink(configmap["sync_folder"].(string), fmt.Sprintf("%s/lpmx", configmap["dir"].(string)))
				if serr != nil {
					cerr := ErrNew(serr, fmt.Sprintf("could not symlink, oldpath: %s, newpath: %s", configmap["sync_folder"].(string), fmt.Sprintf("%s/lpmx", configmap["dir"].(string))))
					return cerr
				}

				//patch ld.so
				ld_new_path := fmt.Sprintf("%s/ld.so.patch", rootfolder)
				LOGGER.WithFields(logrus.Fields{
					"ld_patched_path": ld_new_path,
				}).Debug("layers sha256 list")
				if !FileExist(ld_new_path) {
					for _, v := range LD {
						for _, l := range strings.Split(configmap["layers"].(string), ":") {
							ld_orig_path := fmt.Sprintf("%s/%s%s", configmap["baselayerpath"].(string), l, v)

							LOGGER.WithFields(logrus.Fields{
								"ld_path": ld_orig_path,
							}).Debug("layers sha256 list")
							if _, err := os.Stat(ld_orig_path); err == nil {
								err := Patchldso(ld_orig_path, ld_new_path)
								if err != nil {
									return err
								}
								configmap["elf_loader"] = ld_new_path
								break
							}
						}
						if _, ok := configmap["elf_loader"]; ok {
							break
						}
					}
				} else {
					configmap["elf_loader"] = ld_new_path
				}

				//add current user to /etc/passwd user gid to /etc/group
				user, err := user.Current()
				if err != nil {
					cerr := ErrNew(err, "can't get current user info")
					return cerr
				}

				uname := user.Username
				uid := user.Uid
				gid := user.Gid
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
								return cerr
							}
							defer f.Close()
							_, err = f.WriteString(fmt.Sprintf("%s:x:%s:%s:%s:/home/%s:/bin/bash\n", uname, uid, uid, uname, uname))
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/passwd", new_passwd_path))
								return cerr
							}
						} else {
							return c_err
						}
					}

					group_path := fmt.Sprintf("%s/%s/etc/group", configmap["baselayerpath"].(string), l)
					if _, err := os.Stat(group_path); err == nil {
						new_group_path := fmt.Sprintf("%s/etc", configmap["dir"].(string))
						os.MkdirAll(new_group_path, os.FileMode(FOLDER_MODE))
						ret, c_err := CopyFile(group_path, fmt.Sprintf("%s/group", new_group_path))
						if ret && c_err == nil {
							f, err := os.OpenFile(fmt.Sprintf("%s/group", new_group_path), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/group", new_group_path))
								return cerr
							}
							defer f.Close()
							_, err = f.WriteString(fmt.Sprintf("%s:x:%s\n", uname, gid))
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("%s/group", new_group_path))
								return cerr
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
						} else {
							return c_err
						}
					}
				}

				//create rw/proc/self/cwd to fake cwd
				proc_self_path := fmt.Sprintf("%s/proc/self", configmap["dir"].(string))
				os.MkdirAll(proc_self_path, os.FileMode(FOLDER_MODE))
				os.Symlink("/", fmt.Sprintf("%s/cwd", proc_self_path))
				os.Symlink("/", fmt.Sprintf("%s/exe", proc_self_path))

				//create tmp folder and create whiteout file for tmp
				os.MkdirAll(fmt.Sprintf("%s/tmp", configmap["dir"].(string)), os.FileMode(FOLDER_MODE))
				f, _ := os.Create(fmt.Sprintf("%s/.wh.tmp", configmap["dir"].(string)))
				f.Close()

				//run container
				r_err := Run(&configmap)
				if r_err != nil {
					return r_err
				}
				return nil
			}
		}
		cerr := ErrNew(ErrNExist, fmt.Sprintf("image %s doesn't exist", name))
		return cerr
	}
	if err.Err == ErrNExist {
		err.AddMsg(fmt.Sprintf("image %s does not exist, you may need to download it firstly", name))
	}
	return err
}

func DockerDelete(name string) *Error {
	currdir, _ := GetCurrDir()
	//check if there are containers assocated with current image
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)
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

	rootdir = fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err = unmarshalObj(rootdir, &doc)
	if err == nil {
		if val, ok := doc.Images[name]; ok {
			if vval, vok := val.(map[string]interface{}); vok {
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

func Expose(id string, name string) *Error {
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)

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
						if !strings.Contains(con.ExposeExe, name) {
							if con.ExposeExe == "" {
								con.ExposeExe = name
							} else {
								con.ExposeExe = fmt.Sprintf("%s:%s", con.ExposeExe, name)
							}
						}

						bindir := fmt.Sprintf("%s/bin", currdir)
						if !FolderExist(bindir) {
							_, err := MakeDir(bindir)
							if err != nil {
								return err
							}
						}

						bname := filepath.Base(name)
						bdir := fmt.Sprintf("%s/%s", bindir, bname)
						if FileExist(bdir) {
							RemoveFile(bdir)
						}
						f, ferr := os.OpenFile(bdir, os.O_RDWR|os.O_CREATE, 0755)
						if ferr != nil {
							cerr := ErrNew(ferr, fmt.Sprintf("can not create exposed file %s", bdir))
							return cerr
						}
						code := "#!/bin/bash\n" +
							"lpmx resume " + id + " \"" + name + " " + "\"$@\"\"" +
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

func (con *Container) genEnv() (map[string]string, *Error) {
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
	//used for faking proc file
	env["FAKECHROOT_EXCLUDE_PROC_PATH"] = "/proc/self/cwd:/proc/self/exe"
	if con.DockerBase {
		env["DockerBase"] = "TRUE"
	} else {
		env["DockerBase"] = "FALSE"
	}
	env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	//set default LD_LIBRARY_LPMX
	var libs []string
	//add libmemcached and other libs
	currdir, _ := GetCurrDir()
	libs = append(libs, fmt.Sprintf("%s/.lpmxsys", currdir))

	for _, v := range LD_LIBRARY_PATH_DEFAULT {
		lib_paths, err := GuessPathsContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v, false)
		if err != nil {
			continue
		} else {
			libs = append(libs, lib_paths...)
		}
	}

	for _, v := range LD_LIBRARY_PATH_DEFAULT {
		libs = append(libs, fmt.Sprintf("%s/%s", con.RootPath, v))
	}

	if len(libs) > 0 {
		env["LD_LIBRARY_LPMX"] = strings.Join(libs, ":")
		env["LD_LIBRARY_PATH"] = fmt.Sprintf("%s/.lpmxsys", currdir)
	}

	//set default FAKECHROOT_EXCLUDE_PATH
	env["FAKECHROOT_EXCLUDE_PATH"] = "/dev:/proc:/sys"

	//set data sync folder
	env["FAKECHROOT_DATA_SYNC"] = con.DataSyncFolder

	//set default FAKECHROOT_CMD_SUBSET
	env["FAKECHROOT_CMD_SUBST"] = "/sbin/ldconfig.real=/bin/true:/sbin/insserv=/bin/true:/sbin/ldconfig=/bin/true:/usr/bin/ischroot=/bin/true:/usr/bin/mkfifo=/bin/true"

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
									if con.DockerBase {
										env[k1], err = GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v1, false)
									} else {
										env[k1], err = GuessPath(con.RootPath, v1, false)
									}
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
									if con.DockerBase {
										vv1_abs, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), vv1.(string), false)
										if err != nil {
											continue
										}
										libs = append(libs, vv1_abs)
									} else {
										vv1_abs, err := GuessPath(con.RootPath, vv1.(string), false)
										if err != nil {
											continue
										}
										libs = append(libs, vv1_abs)
									}
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
								if con.DockerBase {
									env[k], err = GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v1, false)
								} else {
									env[k], err = GuessPath(con.RootPath, v1, false)
								}
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
									if con.DockerBase {
										vv1_abs, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), vv1.(string), false)
										if err != nil {
											continue
										}
										libs = append(libs, vv1_abs)
									} else {
										vv1_abs, err := GuessPath(con.RootPath, vv1.(string), false)
										if err != nil {
											continue
										}
										libs = append(libs, vv1_abs)
									}
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

	if _, l_switch_ok := con.SettingConf["fakechroot_log_switch"]; l_switch_ok {
		env["FAKECHROOT_LOG_SWITCH"] = "TRUE"
	} else {
		env["FAKECHROOT_LOG_SWITCH"] = "FALSE"
	}

	if l_level, l_level_ok := con.SettingConf["fakechroot_log_level"]; l_level_ok {
		switch l_level {
		case "DEBUG":
			env["FAKECHROOT_LOG_LEVEL"] = "0"
		case "INFO":
			env["FAKECHROOT_LOG_LEVEL"] = "1"
		case "WARN":
			env["FAKECHROOT_LOG_LEVEL"] = "2"
		case "ERROR":
			env["FAKECHROOT_LOG_LEVEL"] = "3"
		case "FATAL":
			env["FAKECHROOT_LOG_LEVEL"] = "4"
		default:
			env["FAKECHROOT_LOG_LEVEL"] = "3"
		}
	}
	if _, priv_switch_ok := con.SettingConf["fakechroot_priv_switch"]; priv_switch_ok {
		env["FAKECHROOT_PRIV_SWITCH"] = "TRUE"
	}

	if _, fakechroot_debug_ok := con.SettingConf["fakechroot_debug"]; fakechroot_debug_ok {
		env["FAKECHROOT_DEBUG"] = "TRUE"
	}

	if ldso_path, ldso_ok := con.SettingConf["fakechroot_elfloader"]; ldso_ok {
		elfloader_path, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), ldso_path.(string), true)
		if err != nil {
			return nil, err
		}
		env["FAKECHROOT_ELFLOADER"] = elfloader_path
	}

	//set language

	return env, nil
}

func (con *Container) bashShell(args ...string) *Error {
	env, err := con.genEnv()

	if err != nil {
		return err
	}

	if FolderExist(con.RootPath) {
		//we need to start faked-sysv firstly
		faked_sysv := fmt.Sprintf("%s/faked-sysv", con.SysDir)
		foutput, ferr := Command(faked_sysv)
		if ferr != nil {
			return ferr
		}
		faked_str := strings.Split(foutput, ":")
		env["FAKEROOTKEY"] = faked_str[0]

		defer func() {
			fmt.Sprintf("cleanning up faked-sysv with pid: %s\n", faked_str[1])
			KillProcessByPid(faked_str[1])
		}()

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
			if con.DockerBase {
				shpath, err := GuessPathContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), strsh, true)
				if err != nil {
					return err
				}
				con.UserShell = shpath
			} else {
				con.UserShell = filepath.Join(con.RootPath, strsh)
			}
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
	currdir, _ := GetCurrDir()
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := unmarshalObj(rootdir, &sys)

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
			cmap["DockerBase"] = strconv.FormatBool(con.DockerBase)
			cmap["Image"] = con.ImageBase
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
		case *Docker:
			err = StructUnmarshal(data, inf.(*Docker))
		case *DockerInfo:
			err = StructUnmarshal(data, inf.(*DockerInfo))
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

func setMap(id string, tp string, name string, value string, server string) *Error {
	mem, err := MInitServers(server)
	if err != nil {
		return err
	}

	if tp == ELFOP[4] {
		err := mem.MUpdateStrValue(fmt.Sprintf("map:%s:%s", id, name), value)
		if err != nil {
			return err
		}
	} else {
		err := mem.MDeleteByKey(fmt.Sprintf("map:%s:%s", id, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func getMap(id string, name string, server string) (string, *Error) {
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

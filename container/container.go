package container

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
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
	. "github.com/jasonyangshadow/lpmx/rpc"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	RUNNING = iota
	STOPPED

	IDLENGTH = 10
)

var (
	ELFOP                   = []string{"add_needed", "remove_needed", "add_rpath", "remove_rpath", "change_user", "add_allow_priv", "remove_allow_priv", "add_deny_priv", "remove_deny_priv", "add_map", "remove_map"}
	STATUS                  = []string{"RUNNING", "STOPPED"}
	LD                      = []string{"/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2", "/lib/ld.so", "/lib64/ld-linux-x86-64.so.2", "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.1", "/lib64/ld-linux-x86-64.so.1", "/lib/ld-linux.so.2", "/lib/ld-linux.so.1"}
	LD_LIBRARY_PATH_DEFAULT = []string{"lib", "lib/x86_64-linux-gnu", "usr/lib/x86_64-linux-gnu", "usr/lib", "usr/local/lib"}
	FOLDER_MODE             = 0755
)

type Sys struct {
	RootDir      string // the abs path of folder .lpmxsys
	BinaryDir    string // the folder cotnainers .lpmxsys and binaries
	Containers   map[string]interface{}
	LogPath      string
	MemcachedPid string
}

type Container struct {
	Id                  string
	RootPath            string
	ConfigPath          string
	ImageBase           string
	DockerBase          bool
	Layers              string
	BaseLayerPath       string
	Status              int
	LogPath             string
	ElfPatcherPath      string
	PatchedELFLoader    string
	FakechrootPath      string
	SettingConf         map[string]interface{}
	SettingPath         string
	SysDir              string //dir of lpmx set by appendToSys function
	StartTime           string
	ContainerName       string
	CreateUser          string
	CurrentUser         string
	MemcachedServerList []string
	ExposeExe           string
	ShmFiles            string
	IpcFiles            string
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

type Docker struct {
	RootDir string
	Images  map[string]interface{}
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
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	config := fmt.Sprintf("%s/.lpmxsys", currdir)
	sys.RootDir = config
	sys.BinaryDir = currdir
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
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)
	if err == nil {
		fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s", "ContainerID", "Status", "RPC", "DockerBased", "Image"))
		for k, v := range sys.Containers {
			if cmap, ok := v.(map[string]interface{}); ok {
				port := strings.TrimSpace(cmap["RPCPort"].(string))
				if port != "0" {
					conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), time.Millisecond*200)
					if err == nil && conn != nil {
						conn.Close()
						fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s%", k, cmap["Status"].(string), cmap["RPCPort"].(string), cmap["DockerBase"].(string), cmap["ImageBase"].(string)))
					}
				} else {
					fmt.Println(fmt.Sprintf("%s%15s%15s%15s%15s", k, cmap["Status"].(string), "NA", cmap["DockerBase"].(string), cmap["ImageBase"].(string)))
				}
			} else {
				cerr := ErrNew(ErrType, "sys.Containers type error")
				return cerr
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

func Resume(id string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)
	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				if val["Status"].(string) == STATUS[1] {
					configmap := make(map[string]interface{})
					configmap["dir"] = val["RootPath"].(string)
					configmap["config"] = val["SettingPath"].(string)
					configmap["passive"] = false
					if b, _ := strconv.ParseBool(val["DockerBase"].(string)); b {
						configmap["docker"] = true
						configmap["layers"] = val["Layers"].(string)
						configmap["id"] = val["Id"].(string)
						configmap["image"] = val["Image"].(string)
						configmap["baselayerpath"] = val["BaseLayerPath"].(string)
						configmap["elf_loader"] = val["PatchedELFLoader"].(string)
					}
					err := Run(&configmap)
					if err != nil {
						return err
					}
				} else {
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running, can't resume", id))
					return cerr
				}
			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}
	return err

}

func Destroy(id string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
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
					cerr := ErrNew(ErrExist, fmt.Sprintf("conatiner with id: %s is running, can't destroy", id))
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
	return err

}

func Run(configmap *map[string]interface{}) *Error {
	dir, _ := (*configmap)["dir"].(string)
	config, _ := (*configmap)["config"].(string)
	passive, _ := (*configmap)["passive"].(bool)

	rootdir := fmt.Sprintf("%s/.lpmx", dir)

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
		}
	} else {
		con.DockerBase = false
	}
	con.RootPath = dir
	con.ConfigPath = rootdir
	con.SettingPath = config

	defer func() {
		data, _ := StructMarshal(&con)
		WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))
		con.Status = STOPPED
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
		con.Status = RUNNING
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
		con.Status = RUNNING
		err := con.appendToSys()
		if err != nil {
			err.AddMsg("append to sys info error")
			return err
		}
		err = con.bashShell()
		if err != nil {
			err.AddMsg("starting bash shell encounters error")
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

func Get(id string, name string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

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
	return err
}

func Set(id string, tp string, name string, value string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				tp = strings.ToLower(strings.TrimSpace(tp))
				switch tp {
				case ELFOP[9], ELFOP[10]:
					{
						err := setMap(id, tp, name, value, sys.MemcachedPid)
						if err != nil {
							return err
						}
					}
				case ELFOP[5], ELFOP[6]:
					{
						err := setPrivilege(id, tp, name, value, sys.MemcachedPid, true)
						if err != nil {
							return err
						}
					}
				case ELFOP[7], ELFOP[8]:
					{
						err := setPrivilege(id, tp, name, value, sys.MemcachedPid, false)
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
							return cerr
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
								return err_new
							}
							if strings.TrimSpace(name) != "user" {
								err_new := ErrNew(ErrType, "name should be 'user'")
								return err_new
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
								return err_new
							}
						default:
							err_new := ErrNew(ErrType, "tp should be one of 'add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user'}")
							return err_new
						}

						//write back
						data, _ := StructMarshal(&con)
						WriteToFile(data, fmt.Sprintf("%s/.info", con.ConfigPath))

					}

					return nil
				default:
					err_new := ErrNew(ErrType, "tp should be one of 'add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user', 'add_allow_priv','remove_allow_priv','add_deny_priv','remove_deny_priv','add_map','remove_map'}")
					return err_new
				}

			}
		} else {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("conatiner with id: %s doesn't exist", id))
			return cerr
		}
		return nil
	}
	return err
}

func DockerSearch(name string) ([]string, *Error) {
	tags, err := ListTags("", "", name)
	return tags, err
}

func DockerDownload(name string, user string, pass string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := readDocker(rootdir, &doc)
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
		tdata := strings.Split(name, ":")
		tname := tdata[0]
		ttag := tdata[1]
		mdata := make(map[string]interface{})
		mdata["rootdir"] = fmt.Sprintf("%s/%s/%s", doc.RootDir, tname, ttag)
		mdata["config"] = fmt.Sprintf("%s/setting.yml", mdata["rootdir"])
		mdata["image"] = fmt.Sprintf("%s/image", mdata["rootdir"])
		image_dir, _ := mdata["image"].(string)

		//download lyaers
		ret, layer_order, err := DownloadLayers(user, pass, tname, ttag, image_dir)
		if err != nil {
			return err
		}
		mdata["layer"] = ret
		mdata["layer_order"] = strings.Join(layer_order, ":")

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
		err = DownloadSetting(tname, ttag, rdir)
		if err != nil {
			LOGGER.WithFields(logrus.Fields{
				"err":    err,
				"toPath": rdir,
			}).Error("Download setting failure, you may need to manually put setting.yml to 'toPath'")
		}

		//add map to this image
		doc.Images[name] = mdata

		ddata, _ := StructMarshal(doc)
		err = WriteToFile(ddata, fmt.Sprintf("%s/.docinfo", doc.RootDir))
		if err != nil {
			return err
		}
		return nil
	}
}

func DockerList() *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := readDocker(rootdir, &doc)
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
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := readDocker(rootdir, &doc)
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

func DockerCreate(name string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := readDocker(rootdir, &doc)
	if err == nil {
		if val, ok := doc.Images[name]; ok {
			if vval, vok := val.(map[string]interface{}); vok {
				base, _ := vval["base"].(string)
				workspace, _ := vval["workspace"].(string)
				config, _ := vval["config"].(string)
				layers, _ := vval["layer_order"].(string)
				//randomly generate id
				id := RandomString(IDLENGTH)
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
	return err
}

func DockerDelete(name string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	rootdir := fmt.Sprintf("%s/.docker", currdir)
	var doc Docker
	err := readDocker(rootdir, &doc)
	if err == nil {
		if val, ok := doc.Images[name]; ok {
			if vval, vok := val.(map[string]interface{}); vok {
				dir, _ := vval["rootdir"].(string)
				rok, rerr := RemoveAll(dir)
				if rok {
					delete(doc.Images, name)
					ddata, _ := StructMarshal(doc)
					err = WriteToFile(ddata, fmt.Sprintf("%s/.docinfo", doc.RootDir))
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

func DockerExpose(id string, name string) *Error {
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

	if err == nil {
		if v, ok := sys.Containers[id]; ok {
			if val, vok := v.(map[string]interface{}); vok {
				var con Container
				info := fmt.Sprintf("%s/.lpmx/.info", val["RootPath"].(string))
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
						if !FileExist(bdir) {
							f, err := os.OpenFile(bdir, os.O_RDWR|os.O_CREATE, 0755)
							if err != nil {
								cerr := ErrNew(err, fmt.Sprintf("can not create exposed file %s", bdir))
								return cerr
							}
							defer f.Close()
						}

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
	env["LD_PRELOAD"] = fmt.Sprintf("%s/libfakechroot.so %s/libfakeroot-sysv.so", con.FakechrootPath, con.FakechrootPath)
	env["MEMCACHED_PID"] = con.MemcachedServerList[0]
	env["TERM"] = "xterm"
	env["SHELL"] = con.UserShell
	env["ContainerLayers"] = con.Layers
	env["ContainerBasePath"] = con.BaseLayerPath
	env["FAKECHROOT_ELFLOADER"] = con.PatchedELFLoader
	env["PWD"] = "/"
	//used for faking proc file
	env["FAKECHROOT_EXCLUDE_PROC_PATH"] = "/proc/self/cwd:/proc/self/exe"
	if con.DockerBase {
		env["DockerBase"] = "TRUE"
	} else {
		env["DockerBase"] = "FALSE"
	}

	//set default LD_LIBRARY_LPMX
	var libs []string
	//add libmemcached and other libs
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	libs = append(libs, currdir)

	for _, v := range LD_LIBRARY_PATH_DEFAULT {
		lib_paths, err := GuessPathsContainer(filepath.Dir(con.RootPath), strings.Split(con.Layers, ":"), v, false)
		if err != nil {
			continue
		} else {
			libs = append(libs, lib_paths...)
		}
	}

	if len(libs) > 0 {
		env["LD_LIBRARY_LPMX"] = strings.Join(libs, ":")
	}

	//set default FAKECHROOT_EXCLUDE_PATH
	env["FAKECHROOT_EXCLUDE_PATH"] = "/tmp:/dev:/proc:/sys"

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

	if _, l_switch_ok := con.SettingConf["__log_switch"]; l_switch_ok {
		env["__LOG_SWITCH"] = "TRUE"
	} else {
		env["__LOG_SWITCH"] = "FALSE"
	}

	if l_level, l_level_ok := con.SettingConf["__log_level"]; l_level_ok {
		switch l_level {
		case "DEBUG":
			env["__LOG_LEVEL"] = "0"
		case "INFO":
			env["__LOG_LEVEL"] = "1"
		case "WARN":
			env["__LOG_LEVEL"] = "2"
		case "ERROR":
			env["__LOG_LEVEL"] = "3"
		case "FATAL":
			env["__LOG_LEVEL"] = "4"
		default:
			env["__LOG_LEVEL"] = "3"
		}
	}
	if _, priv_switch_ok := con.SettingConf["__priv_switch"]; priv_switch_ok {
		env["__PRIV_SWITCH"] = "TRUE"
	} else {
		env["__PRIV_SWITCH"] = "TRUE"
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
	return env, nil
}

func (con *Container) bashShell() *Error {
	env, err := con.genEnv()
	LOGGER.WithFields(logrus.Fields{
		"env": env,
		"err": err,
	}).Debug("genEnv debug")

	if err != nil {
		return err
	}

	if FolderExist(con.RootPath) {
		LOGGER.WithFields(logrus.Fields{
			"con.userShell": con.UserShell,
			"env":           env,
			"con.RootPath":  con.RootPath,
		}).Debug("shell env paramters")
		err = ShellEnv(con.UserShell, env, con.RootPath)
		if err != nil {
			return err
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
	con.ElfPatcherPath = fmt.Sprintf("%s/elf", con.ConfigPath)
	con.FakechrootPath = fmt.Sprintf("%s/fakechroot", con.ConfigPath)
	user, err := Command("whoami")
	if err != nil {
		return err
	}
	con.CreateUser = strings.TrimSuffix(user, "\n")
	con.V, con.SettingConf, err = LoadConfig(con.SettingPath)
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
	_, err = MakeDir(con.ElfPatcherPath)
	if err != nil {
		return err
	}
	_, err = MakeDir(con.FakechrootPath)
	if err != nil {
		return err
	}

	//find these follwing libraries and binaries in current
	lpmxdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	searchPaths := []string{lpmxdir, "."}

	elf_copied := false
	for _, path := range searchPaths {
		elf_tmp := fmt.Sprintf("%s/patchelf", path)
		if FileExist(elf_tmp) {
			_, err = CopyFile(elf_tmp, fmt.Sprintf("%s/patchelf", con.ElfPatcherPath))
			if err != nil {
				return err
			}
			elf_copied = true
			break
		}
	}
	if !elf_copied {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("can not copy patchelf binary, please put it inside the same foler of lpmx or current folder"))
		return cerr
	}

	libfakechroot_copied := false
	for _, path := range searchPaths {
		libfakechroot_tmp := fmt.Sprintf("%s/libfakechroot.so", path)
		if FileExist(libfakechroot_tmp) {
			_, err = CopyFile(libfakechroot_tmp, fmt.Sprintf("%s/libfakechroot.so", con.FakechrootPath))
			if err != nil {
				return err
			}
			libfakechroot_copied = true
			break
		}
	}
	if !libfakechroot_copied {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("can not copy libfakechroot.so library, please put it inside the same foler of lpmx or current folder"))
		return cerr
	}

	libfakeroot_copied := false
	for _, path := range searchPaths {
		libfakeroot_tmp := fmt.Sprintf("%s/libfakeroot-sysv.so", path)
		if FileExist(libfakeroot_tmp) {
			_, err = CopyFile(libfakeroot_tmp, fmt.Sprintf("%s/libfakeroot-sysv.so", con.FakechrootPath))
			if err != nil {
				return err
			}
			libfakeroot_copied = true
			break
		}
	}
	if !libfakeroot_copied {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("can not copy libfakeroot-sysv.so library, please put it inside the same foler of lpmx or current folder"))
		return cerr
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
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	var sys Sys
	rootdir := fmt.Sprintf("%s/.lpmxsys", currdir)
	err := readSys(rootdir, &sys)

	if err == nil {
		if val, ok := sys.Containers[con.Id]; ok {
			if cmap, cok := val.(map[string]interface{}); cok {
				cmap["Status"] = STATUS[con.Status]
				cmap["RPCPort"] = fmt.Sprintf("%d", con.RPCPort)
			} else {
				cerr := ErrNew(ErrType, "interface{} type assertation error")
				return cerr
			}
		} else {
			cmap := make(map[string]string)
			cmap["Status"] = STATUS[con.Status]
			cmap["RootPath"] = con.RootPath
			cmap["SettingPath"] = con.SettingPath
			cmap["RPCPort"] = fmt.Sprintf("%d", con.RPCPort)
			cmap["DockerBase"] = strconv.FormatBool(con.DockerBase)
			cmap["ImageBase"] = con.ImageBase
			cmap["Layers"] = con.Layers
			cmap["Id"] = con.Id
			cmap["Image"] = con.ImageBase
			cmap["BaseLayerPath"] = con.BaseLayerPath
			cmap["PatchedELFLoader"] = con.PatchedELFLoader
			sys.Containers[con.Id] = cmap
		}
		sys.MemcachedPid = fmt.Sprintf("%s/.memcached.pid", currdir)
		servers := []string{sys.MemcachedPid}
		con.MemcachedServerList = servers
		con.SysDir = sys.BinaryDir
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
		return nil, cerr
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
		return cerr
	}
	return nil
}

func readDocker(rootdir string, doc *Docker) *Error {
	info := fmt.Sprintf("%s/.docinfo", rootdir)
	if FileExist(info) {
		data, err := ReadFromFile(info)
		if err != nil {
			return err
		}
		err = StructUnmarshal(data, doc)
		if err != nil {
			return err
		}
	} else {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s/.docinfo doesn't exist", rootdir))
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
		if tp == ELFOP[5] {
			err := mem.MUpdateStrValue(fmt.Sprintf("allow:%s:%s", id, name), value)
			if err != nil {
				return err
			}
		}
		if tp == ELFOP[6] {
			err := mem.MDeleteByKey(fmt.Sprintf("allow:%s:%s", id, name))
			if err != nil {
				return err
			}
		}
	} else {
		if tp == ELFOP[7] {
			err := mem.MUpdateStrValue(fmt.Sprintf("deny:%s:%s", id, name), value)
			if err != nil {
				return err
			}
		}
		if tp == ELFOP[8] {
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

	if tp == ELFOP[9] {
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

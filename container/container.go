package container

import (
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"os"
	"time"
)

const (
	RUNNING = iota
	PAUSE
	STOPPED

	MAX_CONTAINER_COUNT = 1024
)

var (
	AvalableContainerIds = [MAX_CONTAINER_COUNT]int8{0}
)

type Container struct {
	id                    string
	root_path             string
	status                int8
	log_path              string
	elfpatcher_path       string
	fakechroot_path       string
	setting_conf_path     string
	setting_conf          map[string]string
	start_time            Time
	image_name            string
	container_name        string
	create_user           string
	memcached_server_list string
	shm_files             string
	ipc_files             string
}

func findAvailableId() (int8, *Error) {
	for _, num := range AvalableContainerIds {
		if num == 0 {
			return num, nil
		} else if num == 1 {
			continue
		}
	}
	cerr := ErrNew(ErrFull, "No available container id could be generated")
	return -1 & cerr
}

func CreateContainer(cwd string, image_name string) (*Container, *Error) {
	id, err := findAvailableId()
	if err == nil {
		var con Container
		con.id = fmt.Sprintf("container-%s", string(id))
		con.root_path = fmt.Sprintf("%s/%s/instance", cwd, con.id)
		con.status = STOPPED
		con.log_path = fmt.Sprintf("%s/%s/log", cwd, con.id)
		con.elfpatcher_path = fmt.Sprintf("%s/%s/elf/", cwd, con.id)
		con.fakechroot_path = fmt.Sprintf("%s/%s/fakechroot/", cwd, con.id)
		con.setting_conf_path = fmt.Sprintf("%s/%s/settings/", cwd, con.id)
		con.setting_conf = GetMap("setting.yml", []string{con.setting_conf_path})
		con.image_name = image_name
		return &con, nil
	}
	return nil, err
}

func walkfs(con *Container) ([]string, *Error) {
	fileList := []string{}
	err := filepath.Walk(*con.root_path, func(path string, f os.FileInfo, err error) error {
		ftype, err := fileType(path)
		if err != nil {
			return *err
		}
		if ftype == TYPE_REGULAR {
			bpermission, err := filePermission(path, PERM_EXE)
			if err != nil {
				return *err
			}
			fileList = append(fileList, path)
		}
	})
	return fileList, &err
}

func Cmd(cmd string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmd, arg)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", &err
	}
	return out.String(), nil
}

func CmdWithEnv(cmd string, env map[string]string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmd, arg)
	env := ""
	var out bytes.Buffer
	for key, value := range env {
		env = append(env, fmt.Sprintf("%s=%s,", key, value))
	}
	cmd.Env = env
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", &err
	}
	return out.String(), nil
}

func CmdInShell(cmd string, arg ...string) (int16,*Error) {
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin,os.Stderr,os.Stdout}
	process, err := os.StartProcess(cmd, arg, procAttr); err != nil {
		return -1,&err
	}
	return process.Pid, nil
}

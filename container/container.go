package container

import (
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	. "github.com/jasonyangshadow/lpmx/yaml"
	"os"
	"os/exec"
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

type Container struct {
	id                    string
	root_path             string
	status                int8
	log_path              string
	elfpatcher_path       string
	fakechroot_path       string
	setting_conf_path     string
	setting_conf          map[string]interface{}
	start_time            string
	image_name            string
	container_name        string
	create_user           string
	memcached_server_list string
	shm_files             string
	ipc_files             string
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

func CreateContainer(cwd string, image_name string) (*Container, *Error) {
	id, err := findAvailableId()
	if err == nil {
		var con Container
		con.id = fmt.Sprintf("container-%d", id)
		for strings.HasSuffix(cwd, "/") {
			cwd = strings.TrimSuffix(cwd, "/")
		}
		con.root_path = fmt.Sprintf("%s/%s/instance", cwd, con.id)
		con.status = STOPPED
		con.log_path = fmt.Sprintf("%s/%s/log", cwd, con.id)
		con.elfpatcher_path = fmt.Sprintf("%s/%s/elf/", cwd, con.id)
		con.fakechroot_path = fmt.Sprintf("%s/%s/fakechroot/", cwd, con.id)
		con.setting_conf_path = fmt.Sprintf("%s/%s/settings/", cwd, con.id)
		con.setting_conf, _ = GetMap("setting.yml", []string{con.setting_conf_path})
		con.image_name = image_name
		return &con, nil
	}
	return nil, err
}

func Walkfs(con *Container) ([]string, *Error) {
	fileList := []string{}

	err := filepath.Walk(con.root_path, func(path string, f os.FileInfo, err error) error {
		ftype, err := FileType(path)
		if err != nil {
			return err
		}
		if ftype == TYPE_REGULAR {
			_, err := FilePermission(path, PERM_EXE)
			if err != nil {
				return err
			}
			fileList = append(fileList, path)
		}
		return nil
	})
	cerr := ErrNew(err, "walkfs error")
	return fileList, &cerr
}

func Command(cmdStr string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmdStr, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		cerr := ErrNew(err, "cmd running error")
		return "", &cerr
	}
	return out.String(), nil
}

func CommandEnv(cmdStr string, env map[string]string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmdStr, arg...)
	var out bytes.Buffer
	for key, value := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		cerr := ErrNew(err, "commandenv error")
		return "", &cerr
	}
	return out.String(), nil
}

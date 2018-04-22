package paeudo

import (
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"os"
	"os/exec"
)

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

func CommandEnv(cmdStr string, env map[string]string, dir string, arg ...string) (string, *Error) {
	path, err := exec.LookPath(cmdStr)
	var cmd *exec.Cmd
	if err != nil {
		var args []string
		args = append(args, "-c")
		args = append(args, cmdStr)
		for _, a := range arg {
			args = append(args, a)
		}
		cmd = exec.Command("bash", args...)
	} else {
		cmd = exec.Command(path, arg...)
	}
	cmd.Dir = dir
	for key, value := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", key, value))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		cerr := ErrNew(err, string(out))
		return "", &cerr
	} else {
		return string(out), nil
	}
}

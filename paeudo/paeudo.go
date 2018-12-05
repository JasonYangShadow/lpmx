package paeudo

import (
	"bytes"
	"context"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/log"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Command(cmdStr string, arg ...string) (string, *Error) {
	cmd := exec.Command(cmdStr, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		cerr := ErrNew(err, "cmd running error")
		return "", cerr
	}
	return out.String(), nil
}

func CommandEnv(cmdStr string, env map[string]string, dir string, arg ...string) (string, *Error) {
	path, err := exec.LookPath(cmdStr)
	var cmd *exec.Cmd
	if err != nil {
		bashstr := ""
		bashstr += cmdStr
		for _, a := range arg {
			bashstr += " "
			bashstr += a
		}
		cmd = exec.Command("sh", "-c", bashstr)
	} else {
		cmd = exec.Command(path, arg...)
	}
	cmd.Dir = dir
	envstr := ""
	for key, value := range env {
		envstr += fmt.Sprintf("%s=%s,", key, value)
	}
	cmd.Env = append(os.Environ(), envstr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cerr := ErrNew(err, string(out))
		return "", cerr
	} else {
		return string(out), nil
	}
}

func ShellEnv(sh string, env map[string]string, dir string, arg ...string) *Error {
	shpath, err := exec.LookPath(sh)
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("shell: %s doesn't exist", sh))
		return cerr
	} else {
		cmd := exec.Command(shpath, arg...)
		var envstrs []string
		for key, value := range env {
			envstr := fmt.Sprintf("%s=%s", key, value)
			envstrs = append(envstrs, envstr)
		}
		cmd.Env = envstrs
		cmd.Dir = dir
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		LOGGER.WithFields(logrus.Fields{
			"cmd": cmd,
		}).Debug("shell env debug")
		err := cmd.Run()
		if err != nil {
			cerr := ErrNew(err, "cmd running error")
			return cerr
		}
	}
	return nil
}

func ProcessContextEnv(sh string, env map[string]string, dir string, timeout string, arg ...string) (int, *Error) {
	var cmd *exec.Cmd
	shpath, err := exec.LookPath(sh)
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("shell: %s doesn't exist", sh))
		return -1, cerr
	}
	if strings.TrimSpace(timeout) != "" {
		t, terr := time.ParseDuration(timeout)
		if terr != nil {
			cerr := ErrNew(terr, "time parse error")
			return -1, cerr
		}
		ctx, cancel := context.WithTimeout(context.Background(), t)
		defer cancel()
		cmd = exec.CommandContext(ctx, shpath, arg...)
	} else {
		cmd = exec.Command(shpath, arg...)
	}
	var envstrs []string
	for key, value := range env {
		envstr := fmt.Sprintf("%s=%s", key, value)
		envstrs = append(envstrs, envstr)
	}
	cmd.Env = envstrs
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Start()
	if err != nil {
		cerr := ErrNew(err, "cmd running error")
		return -1, cerr
	}
	return cmd.Process.Pid, nil
}

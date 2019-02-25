package paeudo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/JasonYangShadow/lpmx/error"
	. "github.com/JasonYangShadow/lpmx/log"
	. "github.com/JasonYangShadow/lpmx/pid"
	"github.com/sirupsen/logrus"
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

func CommandBash(cmdStr string) (string, *Error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cerr := ErrNew(err, string(out))
		return "", cerr
	} else {
		return string(out), nil
	}
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

func ShellEnvPid(sh string, env map[string]string, dir string, arg ...string) *Error {
	shpath, err := exec.LookPath(sh)
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("shell: %s doesn't exist", sh))
		return cerr
	}
	var args []string
	if len(arg) > 0 {
		args = append(args, "-c")
		for _, ar := range arg {
			args = append(args, ar)
		}
	}

	cmd := exec.Command(shpath, args...)
	var envstrs []string
	for key, value := range env {
		if len(arg) > 0 {
			if key == "FAKECHROOT_EXCLUDE_PATH" {
				value = value + ":/home"
			}
		}
		envstr := fmt.Sprintf("%s=%s", key, value)
		envstrs = append(envstrs, envstr)
	}
	cmd.Env = envstrs
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	LOGGER.WithFields(logrus.Fields{
		"env": envstrs,
	}).Debug("shell env debug")
	err = cmd.Start()
	if err != nil {
		cerr := ErrNew(err, "cmd start error")
		return cerr
	}

	//starting craeting pid file
	pid_file := fmt.Sprintf("%s/container.pid", filepath.Dir(dir))
	cerr := PidCreateByPid(pid_file, cmd.Process.Pid)
	if cerr != nil {
		return cerr
	}
	err = cmd.Wait()
	if err != nil {
		cerr := ErrNew(err, "cmd wait error")
		return cerr
	}
	return nil
}

func ShellEnv(sh string, env map[string]string, dir string, arg ...string) *Error {
	shpath, err := exec.LookPath(sh)
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("shell: %s doesn't exist", sh))
		return cerr
	} else {
		var args []string
		if len(arg) > 0 {
			args = append(args, "-c")
			for _, ar := range arg {
				args = append(args, ar)
			}
		}

		cmd := exec.Command(shpath, args...)
		var envstrs []string
		for key, value := range env {
			if len(arg) > 0 {
				if key == "FAKECHROOT_EXCLUDE_PATH" {
					value = value + ":/home"
				}
			}
			envstr := fmt.Sprintf("%s=%s", key, value)
			envstrs = append(envstrs, envstr)
		}
		cmd.Env = envstrs
		cmd.Dir = dir
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		LOGGER.WithFields(logrus.Fields{
			"env": envstrs,
		}).Debug("shell env debug")
		err := cmd.Run()
		switch err.(type) {
		case *exec.ExitError:
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

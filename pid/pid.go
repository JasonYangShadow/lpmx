package pid

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

func PidValue(pidfile string) (int, *Error) {
	value, err := ioutil.ReadFile(pidfile)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not open file %s", pidfile))
		return -1, cerr
	}

	pid, err := strconv.ParseInt(string(value), 10, 32)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not strconv value: %s", value))
		return -1, cerr
	}

	return int(pid), nil
}

func PidIsActive(pidfile interface{}) (bool, *Error) {
	var pid int
	var err *Error
	switch pidfile.(type) {
	case string:
		pid, err = PidValue(pidfile.(string))
		if err != nil {
			return false, err
		}
	case int:
		pid = pidfile.(int)
	default:
		cerr := ErrNew(ErrMismatch, "pidfile type should be either string or int")
		return false, cerr
	}

	p, perr := os.FindProcess(pid)
	if perr != nil {
		cerr := ErrNew(perr, fmt.Sprintf("pid %d file could not be found in system", pid))
		return false, cerr
	}

	if err := p.Signal(os.Signal(syscall.Signal(0))); err != nil {
		cerr := ErrNew(err, fmt.Sprintf("send signal to pidfile %s with error %s", pidfile, err.Error()))
		return false, cerr
	}

	return true, nil
}

func PidCreate(pidfile string) (int, *Error) {
	if _, err := os.Stat(pidfile); !os.IsNotExist(err) {
		if pid, perr := PidValue(pidfile); perr != nil {
			return -1, perr
		} else {
			if aok, _ := PidIsActive(pid); aok {
				return pid, nil
			}
		}
	}

	if pf, err := os.OpenFile(pidfile, os.O_RDWR|os.O_CREATE, 0600); err != nil {
		return -1, ErrNew(err, fmt.Sprintf("could not create pidfile: %s", pidfile))
	} else {
		pid := os.Getpid()
		pf.Write([]byte(strconv.Itoa(pid)))
		return pid, nil
	}
}

func PidCreateByPid(pidfile string, pid int) *Error {
	if _, err := os.Stat(pidfile); !os.IsNotExist(err) {
		t_pid, perr := PidValue(pidfile)
		if perr != nil {
			return perr
		}
		if aok, _ := PidIsActive(t_pid); aok {
			if t_pid != pid {
				cerr := ErrNew(ErrPidLive, fmt.Sprintf("%s pid file is still occupied by pid: %d", pidfile, t_pid))
				return cerr
			}
			return nil
		}
	}

	if pf, err := os.OpenFile(pidfile, os.O_RDWR|os.O_CREATE, 0600); err != nil {
		return ErrNew(err, fmt.Sprintf("could not create pidfile: %s", pidfile))
	} else {
		pf.Write([]byte(strconv.Itoa(pid)))
		return nil
	}
}

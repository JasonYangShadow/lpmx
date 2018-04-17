package paeudo

import (
	"bufio"
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	"os"
	"os/exec"
	"strings"
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

func PaeudoShell(dir string) *Error {
	if FolderExist(dir) {
		fmt.Print(fmt.Sprintf("%s>> ", dir))
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "exit" {
				break
			}
			cmds := strings.Fields(text)
			val, err := Command(cmds[0], cmds[1:]...)
			if err == nil {
				fmt.Println(val)
			} else {
				fmt.Println(err)

			}
			fmt.Print(fmt.Sprintf("%s>> ", dir))
		}
		return nil
	}
	cerr := ErrNew(ErrNExist, fmt.Sprintf("input folder: %s doesn't exist", dir))
	return &cerr
}

func ContainerPaeudoShell(con *Container) *Error {
	if FolderExist(con.RootPath) {
		fmt.Print(fmt.Sprintf("@%s>> ", con.Id))
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "exit" {
				break
			}
			cmds := strings.Fields(text)
			cmd := fmt.Sprintf("LD_PRELOAD=%s %s", con.FakechrootPath, cmds[0])
			val, err := Command(cmd, cmds[1:]...)
			if err == nil {
				fmt.Println(val)
			} else {
				fmt.Println(err)
			}
			fmt.Print(fmt.Sprintf("@%s>> ", con.Id))
		}
		return nil
	}
	cerr := ErrNew(ErrNExist, fmt.Sprintf("can't locate container root folder %s", con.RootPath))
	return &cerr
}

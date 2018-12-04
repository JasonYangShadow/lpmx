package elf

import (
	"bytes"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/utils"
	"io/ioutil"
	"os"
)

const (
	SET_INTERPRETER = iota
	SET_SONAME
	SET_RPATH
	ADD_NEEDED
	REMOVE_RPATH
	REMOVE_NEEDED
	REPLACE_NEEDED
)

var PARAMS = []string{"--set-interpreter", "--set-soname", "--set-rpath", "--add-needed", "--remove-rpath", "--remove-needed", "--replace-needed"}

func elfPatch(elfpath string, arg ...string) (string, *Error) {
	cmd := fmt.Sprintf("%s/patchelf", elfpath)
	out, err := Command(cmd, arg...)
	if err == nil {
		return out, nil
	}
	return "", err
}

func ElfSetInterpreter(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(elfpath, PARAMS[SET_INTERPRETER], lib, prog)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ElfSetSoname(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(elfpath, PARAMS[SET_SONAME], lib, prog)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ElfRPath(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(elfpath, PARAMS[SET_RPATH], lib, prog)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ElfAddNeeded(elfpath string, libs []string, prog string) (bool, *Error) {
	for _, lib := range libs {
		_, err := elfPatch(elfpath, PARAMS[ADD_NEEDED], lib, prog)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ElfRemoveNeeded(elfpath string, libs []string, prog string) (bool, *Error) {
	for _, lib := range libs {
		_, err := elfPatch(elfpath, PARAMS[REMOVE_NEEDED], lib, prog)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ElfRemoveRPath(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(elfpath, PARAMS[REMOVE_RPATH], lib, prog)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ElfReplaceNeeded(elfpath string, lib_old string, lib_new string, prog string) (bool, *Error) {
	_, err := elfPatch(elfpath, PARAMS[REPLACE_NEEDED], lib_old, lib_new, prog)
	if err != nil {
		return false, err
	}
	return true, nil
}

func Patchldso(elfpath string, newpath string) *Error {
	content, err := ioutil.ReadFile(elfpath)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("elf file doesn't exist %s", elfpath))
		return cerr
	}
	permissions, p_err := GetFilePermission(elfpath)
	if p_err != nil {
		return p_err
	}

	nul_etc := []byte("\x00/\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	nul_etc1 := []byte("\x00/\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	nul_lib := []byte("\x00/\x00\x00\x00")
	nul_usr := []byte("\x00/\x00\x00\x00\x00\x00\x00\x00")

	etc := []byte("\x00/etc/ld.so.preload\x00")
	etc1 := []byte("\x00/etc/ld.so.cache\x00")
	lib := []byte("\x00/lib")
	usr := []byte("\x00/usr/lib")

	ld_path_orig := []byte("\x00LD_LIBRARY_PATH\x00")
	ld_path_new := []byte("\x00LD_LIBRARY_LPMX\x00")
	content = bytes.Replace(content, etc, nul_etc, -1)
	content = bytes.Replace(content, etc1, nul_etc1, -1)
	content = bytes.Replace(content, lib, nul_lib, -1)
	content = bytes.Replace(content, usr, nul_usr, -1)
	content = bytes.Replace(content, ld_path_orig, ld_path_new, -1)

	err = ioutil.WriteFile(newpath, content, os.FileMode(permissions))
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("ld.so patch failed %s", newpath))
		return cerr
	}
	return nil
}

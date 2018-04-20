package elf

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/paeudo"
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
	fmt.Println(cmd, arg)
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

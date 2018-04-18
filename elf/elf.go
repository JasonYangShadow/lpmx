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

func elfPatch(op int, elfpath string, arg ...string) (string, *Error) {
	flag := PARAMS[op]
	cmd := fmt.Sprintf("%s %s", elfpath, flag)
	out, err := Command(cmd, arg...)
	if err == nil {
		return out, nil
	}
	return "", err
}

func ElfSetInterpreter(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(SET_INTERPRETER, elfpath, lib, prog)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ElfSetSoname(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(SET_SONAME, elfpath, lib, prog)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ElfRPath(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(SET_RPATH, elfpath, lib, prog)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ElfAddNeeded(elfpath string, libs []string, prog string) (bool, *Error) {
	for _, lib := range libs {
		_, err := elfPatch(ADD_NEEDED, elfpath, lib, prog)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ElfRemoveNeeded(elfpath string, libs []string, prog string) (bool, *Error) {
	for _, lib := range libs {
		_, err := elfPatch(REMOVE_NEEDED, elfpath, lib, prog)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ElfRemoveRPath(elfpath string, lib string, prog string) (bool, *Error) {
	_, err := elfPatch(REMOVE_RPATH, elfpath, lib, prog)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ElfReplaceNeeded(elfpath string, lib_old string, lib_new string, prog string) (bool, *Error) {
	_, err := elfPatch(REPLACE_NEEDED, elfpath, lib_old, lib_new, prog)
	if err != nil {
		return false, err
	}
	return true, nil
}

package elf

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	. "github.com/jasonyangshadow/lpmx/error"
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

func elfPatch(op int, con *Container, arg ...string) (string, *Error) {
	flag = PARAMS[int]
	cmd = fmt.Sprintf("%s %s", con.ElfPatcherPath, flag)
	out, err := Command(cmd, arg)
	if err == nil {
		return out, nil
	}
	return "", err
}

func ElfSetInterpreter(con *Container, lib string, prog string) (bool, *Error) {
	out, err := elfPatch(SET_INTERPRETER, con, lib, prog)
	if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func ElfSetSoname(con *Container, lib string, prog string) (bool, *Error) {
	out, err := elfPatch(SET_SONAME, con, lib, prog)
	if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func ElfRPath(con *Container, lib string, prog string) (bool, *Error) {
	out, err := elfPatch(SET_RPATH, con, lib, prog)
	if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func ElfAddNeeded(con *Container, libs []string, prog string) (bool, *Error) {
	for lib := range libs {
		out, err := elfPatch(ADD_NEEDED, con, lib, prog)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ElfRemoveNeeded(con *Container, libs []string, prog string) (bool, *Error) {
	for lib := range libs {
		out, err := elfPatch(REMOVE_NEEDED, con, lib, prog)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ElfRemoveRPath(con *Container, lib string, prog string) (bool, *Error) {
	out, err := elfPatch(REMOVE_RPATH, con, lib, prog)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func ElfReplaceNeeded(con *Container, lib_old string, lib_new string, prog string) (bool, *Error) {
	out, err := elfPatch(REPLACE_NEEDED, con, lib_old, lib_nwe, prog)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

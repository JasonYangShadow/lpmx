package elf

import (
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

func ELFPatch(op int, con *Container, arg ...string) (string, *Error) {
	newarg := []string{con.ElfPatcherPath}
	newarg = append(newarg, arg...)
	out, err := Command(con.ElfPatcherPath, newarg...)
	if err != nil {
		return "", err
	}
	return out, nil
}

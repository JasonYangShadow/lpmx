package paeudo

import (
	"testing"
)

func TestPaeudo1(t *testing.T) {
	env := make(map[string]string)
	dir := "/tmp"
	err := ShellEnv("/bin/bash", env, dir, "ls -al")
	if err != nil {
		t.Error(err)
	}
}

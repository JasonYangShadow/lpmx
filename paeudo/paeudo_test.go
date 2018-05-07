package paeudo

import (
	"testing"
)

func TestPaeudo1(t *testing.T) {
	env := make(map[string]string)
	dir := "/tmp"
	err := ShellEnv("/usr/bin/fakeroot", env, dir, "/usr/bin/bash")
	if err != nil {
		t.Error(err)
	}
}

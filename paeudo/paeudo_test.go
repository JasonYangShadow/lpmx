package paeudo

import (
	"testing"
)

func TestPaeudo1(t *testing.T) {
	env := make(map[string]string)
	env["LD_PRELOAD"] = "test"
	dir := "/tmp/lpmx_test"
	out, err := CommandEnv("cd", env, dir, "..")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(out)
	}
}

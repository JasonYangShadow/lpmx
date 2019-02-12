package pid

import (
	"testing"
)

func TestPid(t *testing.T) {
	pid, err := PidCreate("/tmp/test")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(pid)
	}
}

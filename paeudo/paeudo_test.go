package paeudo

import (
	"testing"
)

func TestPaeudo1(t *testing.T) {
	str, err := CommandBash("ps -ef|grep bash|grep -v grep|awk '{print $2}'")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(str)
	}
}

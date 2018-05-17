package log

import (
	"testing"
)

func TestLog(t *testing.T) {
	log, err := LogNew("")
	if err != nil {
		t.Error(err)
	}
	LogSet(WARNING)
	log.Println(FATAL, "test")
}

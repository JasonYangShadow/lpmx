package log

import (
	"testing"
)

func TestLog(t *testing.T) {
	LogInit("/tmp/log")
	LogError.Println("this is an error msg")
	LogWarning.Println("this is a warning msg")
	LogInfo.Println("this is an info msg")
	LogDebug.Println("this is a debug msg")
	LogFatal.Println("this is a fatal msg")
}

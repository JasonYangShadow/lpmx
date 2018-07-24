package log

import (
	"github.com/sirupsen/logrus"
	"testing"
)

func TestLog(t *testing.T) {
	LOGGER.SetLevel(logrus.DebugLevel)
	LOGGER.Debug("test logrus")
}

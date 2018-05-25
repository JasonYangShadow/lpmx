package log

import (
	"github.com/sirupsen/logrus"
	"os"
)

var (
	LOGGER = logrus.New()
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
}

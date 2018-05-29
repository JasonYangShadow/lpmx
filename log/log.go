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
	level := os.Getenv("LPMX_LOG_LEVEL")
	switch level {
	case "DEBUG":
		LOGGER.SetLevel(logrus.DebugLevel)
	case "INFO":
		LOGGER.SetLevel(logrus.InfoLevel)
	case "WARN":
		LOGGER.SetLevel(logrus.WarnLevel)
	case "ERROR":
		LOGGER.SetLevel(logrus.ErrorLevel)
	case "FATAL":
		LOGGER.SetLevel(logrus.FatalLevel)
	case "PANIC":
		LOGGER.SetLevel(logrus.PanicLevel)
	default:
		LOGGER.SetLevel(logrus.InfoLevel)
	}

}

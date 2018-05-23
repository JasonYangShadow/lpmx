package log

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
	FATAL
)

var (
	CURRENT_LOG_LEVEL = DEBUG
	LOG_STR           = []string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL"}
)

type Log struct {
	logger *log.Logger
}

func LogNew(dir string) (*Log, *Error) {
	l := new(Log)
	err := l.init(dir)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func LogSet(level int) {
	CURRENT_LOG_LEVEL = level
}

func (l *Log) Println(level int, a ...interface{}) {
	if level >= CURRENT_LOG_LEVEL {
		l.logger.SetPrefix(fmt.Sprintf("[LEVEL: %s] ", LOG_STR[level]))
		l.logger.Println(a...)
	}
}

func (l *Log) init(dir string) *Error {
	multiouts := false
	if strings.TrimSpace(dir) != "" {
		multiouts = true
	}
	if multiouts {
		current_date := time.Now().Local()
		file := fmt.Sprintf("%s/log-%s", dir, current_date.Format("2006-01-02"))
		if !FolderExist(dir) {
			_, err := MakeDir(dir)
			if err != nil {
				return err
			}
		}
		fp, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("can't open log file: %s", file))
			return cerr
		}
		l.logger = log.New(io.MultiWriter(fp, os.Stdout), "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		l.logger = log.New(io.MultiWriter(os.Stdout), "", log.Ldate|log.Ltime|log.Lshortfile)
	}
	return nil
}

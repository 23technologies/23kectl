package logger

import (
	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"os"
)

var _logger logr.Logger

func Init() {
	var intLevel logrus.Level
	var err error

	stringLevel := os.Getenv("LOG")

	if stringLevel == "" {
		intLevel = logrus.InfoLevel
	} else {
		intLevel, err = logrus.ParseLevel(stringLevel)
		if err != nil {
			panic(err)
		}
	}

	intLevel = logrus.TraceLevel

	logrusLog := logrus.New()
	logrusLog.SetLevel(intLevel)

	_logger = logrusr.New(logrusLog)
}

func Get(name ...string) logr.Logger {
	if len(name) == 1 {
		return _logger.WithName(name[0])
	} else {
		return _logger
	}
}


package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

var allLogLevels = ""

func init() {
	needSeparator := false
	for _, logLevel := range logrus.AllLevels {
		if needSeparator {
			allLogLevels += "|"
		} else {
			needSeparator = true
		}

		allLogLevels += strings.ToUpper(logLevel.String())
	}
}

func initLogger(lvl logrus.Level) {
	logger = &logrus.Logger{
		Out:   os.Stderr,
		Level: lvl,

		Formatter: &logrus.TextFormatter{
			DisableColors: false,

			DisableLevelTruncation: true,
			PadLevelText:           true,
			DisableSorting:         false,

			FullTimestamp:   true,
			TimestampFormat: "15:04:05.000",
		},
	}
}

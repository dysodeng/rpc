package logger

import (
	"go.uber.org/zap"
)

var defaultLogger *zap.Logger

func Init(production bool) {
	var err error
	if production {
		defaultLogger, err = zap.NewProduction()
	} else {
		defaultLogger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic(err)
	}
}

func Logger() *zap.Logger {
	return defaultLogger
}

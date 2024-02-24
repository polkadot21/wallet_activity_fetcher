package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type Logger struct {
	*logrus.Logger
}

const (
	jsonMode           = "LOG_JSON"
	EnvLevel           = "LOG_LEVEL"
	ServiceName        = ""
	defaultServiceName = "wallet-activity-parser"
)

func New() *Logger {
	inst := logrus.New()

	envLogJson := strings.Trim(strings.ToLower(os.Getenv(jsonMode)), " ")

	logLevel, err := logrus.ParseLevel(os.Getenv(EnvLevel))
	if err != nil {
		inst.Warnf("Logger: %s", err)
		logLevel = logrus.DebugLevel
	}

	if envLogJson == "true" {
		inst.SetFormatter(&logrus.JSONFormatter{})
		inst.Warningln("JSON mode enabled!")
	}

	serviceName := os.Getenv(ServiceName)
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	inst.SetOutput(os.Stdout)
	inst.SetLevel(logLevel)
	inst.WithField("name", serviceName)
	inst.Warningf("Log EnvLevel: %s", logLevel.String())

	return &Logger{
		Logger: inst,
	}
}

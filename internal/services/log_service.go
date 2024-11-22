package services

import (
	"Boxed/internal/config"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LogService struct {
	Log    *logrus.Logger
	config config.Configuration
}

func NewLogService(configuration *config.Configuration) LogService {
	log := logrus.New()
	setLogOutputType(configuration, log)
	setLogLevel(configuration, log)
	setLogFormatter(configuration, log)
	return LogService{
		Log: log,
	}
}

func setLogFormatter(configuration *config.Configuration, log *logrus.Logger) {
	switch configuration.Server.LogConfig.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{})

	}
}

func setLogLevel(configuration *config.Configuration, log *logrus.Logger) {
	switch strings.ToLower(configuration.Server.LogConfig.Level) {
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	case "fatal":
		log.SetLevel(logrus.FatalLevel)
	case "panic":
		log.SetLevel(logrus.PanicLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)

	}
}

func setLogOutputType(configuration *config.Configuration, log *logrus.Logger) {
	outputType := configuration.Server.LogConfig.Output
	switch outputType {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "file":
		if configuration.Server.LogConfig.Output != "" {
			logFolder := strings.TrimRight(configuration.Server.LogConfig.LogPath, "/")
			logName := fmt.Sprintf("%s-%s.log", "boxed", time.Now().Format("2006-01-02"))
			logPath := filepath.Join(logFolder, logName)
			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatal(err)
			}
			log.Out = file
		} else {
			err := fmt.Errorf("file output requires logPath to be set")
			println(err.Error())
		}
	}
}

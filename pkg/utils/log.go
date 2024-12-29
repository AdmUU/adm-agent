/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package utils

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger = logrus.New()

func init() {
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
        FullTimestamp:   true,
        TimestampFormat: "2006-01-02 15:04:05",
    })
}

func GetLogger() *logrus.Logger {
	return logger
}

func SetLoggerLevel() {
    level := logrus.InfoLevel
    env := viper.GetString("app.env")
    if (env == "dev") {
        level = logrus.DebugLevel
    }
	logger.SetLevel(level)
	log.Debugf("Environment: %v", env)
}
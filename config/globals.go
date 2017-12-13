package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deciphernow/object-drive-server/util"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// NodeID is our randomly assigned identifier(on startup node identifier used in zk, logs, etc)
	NodeID = RandomID()

	// RootLogger is from which all other loggers are defined - because this is where we get NodeID in logs
	RootLogger = initLogger()

	// By default, the logger used in config package is the RootLogger
	logger = RootLogger

	// MyIP gives the IP to expose
	MyIP = lookupOurIP()
)

// RandomID generates a random string
func RandomID() string {
	buf := make([]byte, 4)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

// TimeEncoder is a custom encoder for zap.logging in RFC3339Nano format
func TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.RFC3339Nano))
}

// initLogger sets up the logger
func initLogger() *zap.Logger {
	var lvl zapcore.Level
	switch strings.ToUpper(getEnvOrDefault(OD_LOG_LEVEL, "INFO")) {
	case "-1", "DEBUG":
		lvl = zap.DebugLevel
	case "0", "INFO":
		lvl = zap.InfoLevel
	case "1", "WARN":
		lvl = zap.WarnLevel
	case "2", "ERROR":
		lvl = zap.ErrorLevel
	default:
		lvl = zap.InfoLevel
	}
	atomiclvl := zap.NewAtomicLevelAt(lvl)

	cfg := zap.Config{
		Level:       atomiclvl,
		Development: true,
		Encoding:    "console", // currently have to use `console` instead of `json` due to flaws in implementation
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "tstamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	log, err := cfg.Build()
	if err != nil {
		fmt.Println(err)
		panic("logging config had an error")
	}

	return log
}

func lookupOurIP() string {
	return util.GetIP(RootLogger)
}

// RegexEscape is a helper method that takes a string and replaces the period metacharacter with backslash escaping.
func RegexEscape(str string) string {
	return strings.Replace(str, ".", "\\.", -1)
}

// GetEnvOrDefault looks up an environment variable by name.
// If it exists, its value is returned, otherwise a passed in default value is returned
func GetEnvOrDefault(name, defaultValue string) string {
	envVal := os.Getenv(name)
	if len(envVal) == 0 {
		return defaultValue
	}
	return envVal
}

// GetEnvOrDefaultInt looks up an environment variable by name, and returns in integer format
// If it exists, its value is returned in integer format. If not, or an error conversion,
// then passed in default is used.
func GetEnvOrDefaultInt(name string, defaultValue int) int {
	return getEnvOrDefaultIntLogged(RootLogger, name, defaultValue)
}

func getEnvOrDefaultIntLogged(logger *zap.Logger, name string, defaultValue int) int {
	envVal := os.Getenv(name)
	if len(envVal) == 0 {
		return defaultValue
	}
	i, err := strconv.Atoi(envVal)
	if err != nil {
		logger.Info(
			"Environment variable did not parse as an int, so was given a default value",
			zap.String("name", name),
		)
		return defaultValue
	}
	return i
}

package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deciphernow/object-drive-server/util"

	"github.com/uber-go/zap"
)

var (
	// NodeID is our randomly assigned identifier(on startup node identifier used in zk, logs, etc)
	NodeID = RandomID()

	// RootLogger is from which all other loggers are defined - because this is where we get NodeID in logs
	RootLogger = initLogger()

	// By default, the logger used in config package is the RootLogger
	logger = RootLogger

	// We need to know if we are seeing our own IP in some cases
	MyIP = lookupOurIP()
)

// RandomID generates a random string
func RandomID() string {
	buf := make([]byte, 4)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func initLogger() zap.Logger {
	var lvl zap.Option
	switch getEnvOrDefaultInt(OD_LOG_LEVEL, 0) {
	case -1:
		lvl = zap.DebugLevel
	case 0:
		lvl = zap.InfoLevel
	case 1:
		lvl = zap.WarnLevel
	case 2:
		lvl = zap.ErrorLevel
	default:
		lvl = zap.InfoLevel
	}

	// Create a formatter that takes name, zone, example as format
	tf := func() zap.TimeFormatter {
		return zap.TimeFormatter(func(t time.Time) zap.Field {
			return zap.String("tstamp", t.Format(time.RFC3339Nano))
		})
	}

	logger := zap.New(
		zap.NewJSONEncoder(tf()),
		lvl,
		zap.Output(os.Stdout),
		zap.ErrorOutput(os.Stdout)).With(zap.String("node", NodeID))

	return logger
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

func getEnvOrDefaultIntLogged(logger zap.Logger, name string, defaultValue int) int {
	envVal := os.Getenv(name)
	if len(envVal) == 0 {
		return defaultValue
	}
	i, err := strconv.Atoi(envVal)
	if err != nil {
		logger.Warn(
			"Environment variable did not parse as an int, so was given a default value",
			zap.String("name", name),
		)
		return defaultValue
	}
	return i
}

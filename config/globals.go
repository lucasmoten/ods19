package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"decipher.com/object-drive-server/util"

	"github.com/uber-go/zap"
)

var (
	// NodeID is our randomly assigned identifier(on startup node identifier used in zk, logs, etc)
	NodeID = RandomID()

	// RootLogger is from which all other loggers are defined - because this is where we get NodeID in logs
	RootLogger = initLogger()

	// By default, the logger used in config package is the RootLogger
	logger = RootLogger

	// MyIP is used for development only. It overrides the reported lookup of IP based upon the hostname.
	MyIP = lookupOurIP()

	// Port is used for development tests only. It overrides the port used when sending test requests
	// to bypass local NGINX Gatekeeper container for hosts that have issues with docker in some environments
	Port = lookupDockerVMPort()

	// RootURL is the base url for our app - TODO: deprecate this
	RootURL = ""

	// RootURLRegex is the routing url regex for our entire app - TODO: deprecate this
	RootURLRegex = RegexEscape(RootURL)
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

func lookupDockerHost() string {
	answer := "proxier"
	// TODO: Find out why test clients need to use this, and what kinds.
	// TODO: Find out why this necessitates definining a DOCKER_HOST and OD_DOCKERVM_OVERRIDE and what the differences are.
	//This is used by test clients
	dockerhost := os.Getenv("DOCKER_HOST")
	if dockerhost != "" {
		answer = strings.Split(dockerhost, ":")[1][2:]
	}
	return GetEnvOrDefault("OD_DOCKERVM_OVERRIDE", answer)
}

func lookupDockerVMPort() string {
	return GetEnvOrDefault("OD_DOCKERVM_PORT", "8080")
}

func lookupOurIP() string {
	ip := util.GetIP(RootLogger)
	if len(ip) > 0 {
		return ip
	}
	// TODO: Isolate the reason why this hack is in here.
	return lookupDockerHost()
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

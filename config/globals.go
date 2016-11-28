package config

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/uber-go/zap"
)

var (
	// NodeID is our randomly assigned identifier(on startup node identifier used in zk, logs, etc)
	NodeID = RandomID()

	// RootLogger is from which all other loggers are defined - because this is where we get NodeID in logs
	RootLogger = initLogger()

	// DockerVM is used for development tests only. It is the default resolve for the dockervm hostname.
	// Use an IP address to get around DNS resolution issues with docker in some environments
	DockerVM = lookupDockerHost()

	// MyIP is used for development only. It overrides the reported lookup of IP based upon the hostname.
	MyIP = lookupOurIP()

	// Port is used for development tests only. It overrides the port used when sending test requests
	// to bypass local NGINX Gatekeeper container for hosts that have issues with docker in some environments
	Port = lookupDockerVMPort()

	// RootURL is the base url for our app - TODO: deprecate this
	RootURL = ""

	// RootURLRegex is the routing url regex for our entire app - TODO: deprecate this
	RootURLRegex = RegexEscape(RootURL)

	// NginxRootURL should only be refrenced by our generic UI for routing purposes to fill in the base href in templates
	NginxRootURL = "/services/object-drive/1.0"
)

// RandomID generates a random string
func RandomID() string {
	buf := make([]byte, 4)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func initLogger() zap.Logger {
	//We must use NodeID - the same random number used to register in zk
	logger := zap.New(zap.NewJSONEncoder(), zap.Output(os.Stdout), zap.ErrorOutput(os.Stdout)).With(zap.String("node", NodeID))
	logger.SetLevel(zap.Level(getEnvOrDefaultIntLogged(logger, "OD_LOG_LEVEL", 0)))
	return logger
}

func lookupDockerHost() string {
	answer := "dockervm"
	//This is used by test clients
	dockerhost := os.Getenv("DOCKER_HOST")
	if dockerhost != "" {
		answer = strings.Split(dockerhost, ":")[1][2:]
	}
	return GetEnvOrDefault("OD_DOCKERVM_OVERRIDE", answer)
}

func lookupDockerVMPort() string {
	answer := "8080"
	//Allow us to change the port, to get around nginx
	p := GetEnvOrDefault("OD_DOCKERVM_PORT", "8080")
	if p != "" && len(p) > 0 {
		answer = p
	}
	return answer
}

func lookupOurIP() string {
	answer := lookupDockerHost()
	//Find our IP that we want gatekeeper to contact us with
	hostname, err := os.Hostname()
	if err != nil {
		RootLogger.Error("could not look up our own hostname to find ip for gatekeeper")
	}
	if len(hostname) > 0 {
		myIPs, err := net.LookupIP(hostname)
		if err != nil {
			RootLogger.Error("could not get a set of ips for our hostname")
		}
		if len(myIPs) > 0 {
			for a := range myIPs {
				if myIPs[a].To4() != nil {
					answer = myIPs[a].String()
					break
				}
			}
		} else {
			RootLogger.Error("We did not find our ip")
		}
	} else {
		RootLogger.Error("We could not find our hostname")
	}
	return answer
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

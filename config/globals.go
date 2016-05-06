package config

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"decipher.com/object-drive-server/util"
)

// CertsDir is a base certificate directory that expects /server and /client
// to exist inside of it. TODO: Consider the total amount of config we need
// for this project. Is this a sane expectation for all binaries we compile from
// the /cmd subdirectory?
var CertsDir string

// ProjectRoot is a global config variable that corresponds to the base directory.
var ProjectRoot string

// ProjectName is configurable in case the project is migrated to another
// git repository.
var ProjectName = "object-drive-server"

// Set up global configs.
func init() {

	ProjectRoot = locateProjectRoot()
	CertsDir = locateCerts(ProjectRoot)
}

func locateProjectRoot() string {
	var projectRoot string
	var err error

	gopath := GetEnvOrDefault("GOPATH", "")
	if gopath == "" {
		log.Printf("GOPATH is not set. Using current directory for project root.")
		projectRoot, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		projectRoot = filepath.Join(gopath, "src", "decipher.com", ProjectName)
	}

	ok, err := util.PathExists(projectRoot)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Println("ProjectRoot does not exist")
	}
	log.Println("Located project root at", projectRoot)
	return projectRoot
}

func locateCerts(projectRoot string) string {
	var certsDir string
	certsDir = filepath.Join(projectRoot, "defaultcerts")
	ok, err := util.PathExists(certsDir)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Println("Certificates directory does not exist")
	}
	return certsDir
}

// DockerVM is used for development tests only. It is the default resolve for the dockervm hostname.
// Use an IP address to get around DNS resolution issues with docker in some environments
var DockerVM = "dockervm"

// MyIP is used for development only. It overrides the reported lookup of IP based upon the hostname.
var MyIP = "dockervm"

// Port is used for development tests only. It overrides the port used when sending test requests
// to bypass local NGINX Gatekeeper container for hosts that have issues with docker in some environments
var Port = "8080"

// StandaloneMode should be used for development only.  When enabled, it bypasses AAC checks for Get,
// Update based calls, and will not store/retrieve from S3, relying upon a local cache only.
var StandaloneMode = false

func init() {
	//Resolve the dockervm address
	ips, err := net.LookupIP("dockervm")
	if err != nil {
		log.Printf("unable to resolve hostname: dockervm")
	}
	if len(ips) > 0 {
		theIP := ips[0]
		DockerVM = theIP.String()
	}

	DockerVM = GetEnvOrDefault("OD_DOCKERVM_OVERRIDE", DockerVM)

	//Find our IP that we want gatekeeper to contact us with
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("could not lookup hostname")
	}
	if len(hostname) > 0 {
		MyIPs, err := net.LookupIP(hostname)
		if err != nil {
			log.Printf("could not get a set of ips for our hostname")
		}
		if len(MyIPs) > 0 {
			for a := range MyIPs {
				if MyIPs[a].To4() != nil {
					MyIP = MyIPs[a].String()
					break
				}
			}
		} else {
			log.Printf("We did not find our ip")
		}
	} else {
		log.Printf("We could not find our hostname")
	}
	log.Printf("we are %s", MyIP)

	//Allow us to change the port, to get around nginx
	p := GetEnvOrDefault("OD_DOCKERVM_PORT", "8080")
	if p != "" && len(p) > 0 {
		Port = p
	}

	//Allow us to work without a network
	s := GetEnvOrDefault("OD_STANDALONE", "false")
	if s == "true" {
		StandaloneMode = true
	}
}

// RegexEscape is a helper method that takes a string and replaces the period metacharacter with backslash escaping.
func RegexEscape(str string) string {
	return strings.Replace(str, ".", "\\.", -1)
}

// RootURL is the base url for our app
var RootURL = ""

// RootURLRegex is the routing url regex for our entire app
var RootURLRegex = RegexEscape(RootURL)

// NginxRootURL should only be refrenced by our generic UI for routing purposes to fill in the base href in templates
var NginxRootURL = "/services/object-drive/1.0"

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
	envVal := os.Getenv(name)
	if len(envVal) == 0 {
		return defaultValue
	}
	i, err := strconv.Atoi(envVal)
	if err != nil {
		return defaultValue
	}
	return i
}

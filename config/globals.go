package config

import (
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"

	"decipher.com/oduploader/util"
)

// Environment is all parameters passable into this program
// Try to completely eliminate global variables from environment this way
//
// If we are to have multiple programs, they should share as much of the interface
// as possible.
// TODO: Where is this used? And which fields are used?
type Environment struct {
	UsingServerTLS  bool
	HideFileNames   bool
	TCPPort         int
	TCPBind         string
	MasterKey       string
	Partition       string
	BufferSize      int
	KeyBytes        int
	ServerCertFile  string
	ServerKeyFile   string
	ServerTrustFile string
	RsaEncryptBits  int
	AwsConfig       string
	AwsBucket       string
}

// FlagSetup does standard flag setup for this project
func FlagSetup(env *Environment) error {
	//masterkey comes from env to keep it from showing up in ps output
	env.MasterKey = os.Getenv("masterkey")
	certsDir := filepath.Join(ProjectRoot, "defaultcerts")
	flag.StringVar(&env.AwsConfig, "awsConfig", "default", "the config entry to connect to aws")
	flag.BoolVar(&env.HideFileNames, "hideFileNames", true, "use unhashed file and user names")
	flag.IntVar(&env.TCPPort, "tcpPort", 6443, "set the tcp port")
	flag.StringVar(&env.TCPBind, "tcpBind", "0.0.0.0", "tcp bind port")
	flag.StringVar(&env.AwsBucket, "awsBucket", "decipherers", "home bucket to store files in")
	flag.StringVar(&env.Partition, "partition", "partition", "partition within a bucket, and file cache location")
	flag.IntVar(&env.BufferSize, "bufferSize", 1024*4, "the size of a buffer between streams in a session")
	flag.IntVar(&env.KeyBytes, "keyBytes", 32, "AES key size in bytes")
	flag.StringVar(&env.ServerTrustFile, "serverTrustFile", filepath.Join(certsDir, "server", "server.trust.pem"), "The SSL Trust in PEM format for this server")
	flag.StringVar(&env.ServerCertFile, "serverCertFile", filepath.Join(certsDir, "server", "server.cert.pem"), "The SSL Cert in PEM format for this server")
	flag.StringVar(&env.ServerKeyFile, "serverKeyFile", filepath.Join(certsDir, "server", "server.key.pem"), "The private key for the SSL Cert for this server")
	flag.IntVar(&env.RsaEncryptBits, "rsaEncryptBits", 1024, "The number of bits to encrypt a user file key with")
	flag.Parse()
	//Give errors now if the environment is not consistent
	if env.UsingServerTLS {

		_, err := os.Stat(env.ServerTrustFile)
		if err != nil {
			log.Printf("Could not check trust pem %s:%v", env.ServerTrustFile, err)
			return err
		}

		_, err = os.Stat(env.ServerCertFile)
		if err != nil {
			log.Printf("Could not check cert pem %s:%v", env.ServerCertFile, err)
			return err
		}

		_, err = os.Stat(env.ServerKeyFile)
		if err != nil {
			log.Printf("Could not check key pem %s:%v", env.ServerKeyFile, err)
			return err
		}
	}
	return nil
}

// CertsDir is a base certificate directory that expects /server and /client
// to exist inside of it. TODO: Consider the total amount of config we need
// for this project. Is this a sane expectation for all binaries we compile from
// the /cmd subdirectory?
var CertsDir string

// ProjectRoot is a global config variable that corresponds to the base directory.
var ProjectRoot string

// ProjectName is configurable in case the project is migrated to another
// git repository.
var ProjectName = "oduploader"

// Set up global configs.
func init() {

	ProjectRoot = locateProjectRoot()
	CertsDir = locateCerts(ProjectRoot)
}

func locateProjectRoot() string {
	var projectRoot string
	var err error

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Printf("GOPATH is not set.")
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

// The default resolve for "dockervm" hostname.  Pass in an IP
// to get around DNS issues with docker
var DockerVM = "dockervm"
var MyIP = "dockervm"

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
	//Find our IP that we want gatekeeper to contact us with
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("could nost lookup hostname")
	}
	if len(hostname) > 0 {
		MyIPs, err := net.LookupIP(hostname)
		if err != nil {
			log.Printf("could not get a set of ips for our hostname")
		}
		if len(MyIPs) > 0 {
			MyIP = MyIPs[0].String()
		} else {
			log.Printf("We did not find our ip")
		}
	} else {
		log.Printf("We could not find our hostname")
	}
	log.Printf("we are %s", MyIP)
}

// RootURLRegex is the routing url regex for our entire app
var RootURLRegex = `/service/object\-drive/1\.0`

// RootURL is the base url for our app
var RootURL = `/service/object-drive/1.0`

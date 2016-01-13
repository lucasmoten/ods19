package config

import (
	"log"
	"os"
	"path/filepath"

	"decipher.com/oduploader/util"
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
		projectRoot = filepath.Join(gopath, "decipher.com", ProjectName)
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

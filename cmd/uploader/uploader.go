package main

import (
	"decipher.com/oduploader/cmd/uploader/libs"
	"decipher.com/oduploader/config"
	"log"
)

/*
  Do dependency injection part of the app here, and call
	out to proper packages to get things started.
*/
func main() {
	//Get a global environment that is sanity checked
	env := &config.Environment{
		UsingServerTLS: true,
		UsingClientTLS: true,
		UsingAWS:       true,
		UsingLog:       true,
	}
	err := config.FlagSetup(env)
	if err != nil {
		log.Printf("FlagSetup is not consistent:%v %v", env, err)
		return
	}

	//A TLS configuration that the uploader will use
	tls := config.NewUploaderTLSConfigWithEnvironment(env)
	//Create an uploader object
	uploader := libs.CreateUploader(env, tls)
	//Generate the server
	server := uploader.CreateUploadServer()

	log.Printf("Launching web server at: %s", server.Addr)
	server.ListenAndServeTLS(env.ServerCertFile, env.ServerKeyFile)
}

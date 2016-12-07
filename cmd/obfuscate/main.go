package main

import (
	"fmt"
	"os"

	"decipher.com/object-drive-server/config"

	"github.com/deciphernow/commons/gov/encryptor"
)

func help() {
	// Note that you can't escape backticks.  So pass in as parameters
	fmt.Printf(
		`usage:

    #Encrypt a key for use in env.sh
    obfuscate encrypt "$OD_ENCRYPT_MASTERKEY"
    #Encrypt a literal value
    obfuscate encrypt myPzW3rd

	#embed into env.sh literally:
	export OD_ENCRYPT_PASSWORD=ENC{...}
	
    #Environment variable OD_TOKENJAR_LOCATION is the full location of token.jar
    #The default is /opt/services/object-drive-1.0/token.jar if not specified
    #If you are in the build environment, then token.jar is at ../../defaultcerts/token.jar relative to the binary

    #Environment variable OD_TOKENJAR_PASSWORD is key used to encode token.jar
    #The default value should be taken under normal circumstances, because
    #the default is generally hardcoded such that an updated object-drive rpm gives the right value
}
`,
	)
}

func main() {

	// This yields the encrypted version of the variable.
	// We no longer trivially provide the decrypt on the command-line.
	// The apps need to decrypt this themselves.
	if len(os.Args) > 2 && os.Args[1] == "encrypt" {
		key, err := config.GetTokenJarKey()
		if err != nil {
			fmt.Printf("unable to get encrypt key: %v", err)
			os.Exit(1)
		}
		name := os.Args[2]
		encryptValue, err := encryptor.EncryptParameter(name, key)
		if err != nil {
			fmt.Printf("unable to encrypt value: %v", err)
			os.Exit(1)
		}
		fmt.Printf("%s", encryptValue)
		os.Exit(0)
	}

	// If we don't match any of these, then display help
	help()
	os.Exit(1)
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"

	"decipher.com/object-drive-server/client"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"github.com/deciphernow/gov-go/testcerts"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "odrive-test-cli"
	app.Usage = "odrive CRUD operations from the command-line for testing"

	// Declare flags common to commands, and pass them in Flags below.
	confFlag := cli.StringFlag{
		Name:  "conf",
		Usage: "Path to yaml config",
	}

	jsonFlag := cli.BoolFlag{
		Name:  "json",
		Usage: "print all responses as formatted JSON",
	}

	app.Commands = []cli.Command{
		{
			Name:  "example-conf",
			Usage: "print an example configuration file",
			Action: func(clictx *cli.Context) error {

				conf := client.Config{
					Cert:       "/path/to/test.cert.pem",
					Trust:      "/path/to/client.trust.pem",
					Key:        "/path/to/test.key.pem",
					SkipVerify: true,
					Remote:     "https://url.to.odrive",
				}

				fmt.Println("# Example configuration file for odrive")
				prettyPrint(conf, "yaml")

				return nil
			},
		},
		{
			Name:  "test-connection",
			Usage: "establish connection to odrive and check for erros",
			Flags: []cli.Flag{confFlag},
			Action: func(clictx *cli.Context) error {

				conf, err := gatherConf(clictx.String("conf"))
				if err != nil {
					log.Println(err)
				}

				c, err := client.NewClient(conf)
				if err != nil {
					log.Println("could not establish connection", err)
				}

				log.Println("connection established successfully", c)
				return nil
			},
		},
		{
			Name:  "upload",
			Usage: "upload file to odrive",
			Flags: []cli.Flag{confFlag, jsonFlag},
			Action: func(clictx *cli.Context) error {
				conf, err := gatherConf(clictx.String("conf"))
				if err != nil {
					log.Println(err)
				}

				c, err := client.NewClient(conf)
				if err != nil {
					log.Println("Could not establish connection", err)
				}

				var permissions = protocol.Permission{
					Read: protocol.PermissionCapability{
						AllowedResources: []string{"user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"},
					}}

				for _, fileName := range clictx.Args() {
					fmt.Printf("uploading %s...", fileName)
					var obj = protocol.CreateObjectRequest{
						TypeName:   "File",
						Name:       fileName,
						RawAcm:     `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
						Permission: permissions,
						OwnedBy:    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
					}

					fReader, err := os.Open(fileName)
					if err != nil {
						log.Println(err)
					}
					newObj, err := c.CreateObject(obj, fReader)
					if err != nil {
						log.Println("create error: ", err)
					}

					fmt.Println("done")

					if clictx.Bool("json") {
						prettyPrint(newObj, "json")
					}

				}

				return nil
			},
		},
	}

	// Global flags. Used when no "command" passed. Must be repeated above for commands.
	app.Flags = []cli.Flag{
		confFlag,
		jsonFlag,
	}

	// There is no "default" command.  Print help and exit.
	app.Action = func(clictx *cli.Context) error {
		fmt.Printf("Must specify command. Run `%s help` for info\n", app.Name)
		return nil
	}

	app.Run(os.Args)
}

// gatherConf prepares the Config object necessary to perform actions in odrive.
// Calling gatherConf with a blank string invokes hard-coded default values and
// certificates, while calling with a named YAML file will load the given values.
// If a YAML file is specified, ALL values must be set.
func gatherConf(confFile string) (client.Config, error) {
	conf := client.Config{}

	if confFile == "" {
		// Get defaults from gov-go binary assets

		// Retrieve Cert
		certContent, err := testcerts.Asset("testcerts/test_0.cert.pem")
		if err != nil {
			log.Println(err)
		}
		certFile, err := writeContents(certContent)
		if err != nil {
			log.Println(err)
		}
		conf.Cert = certFile

		// Retrieve Key
		keyContents, err := testcerts.Asset("testcerts/test_0.key.pem")
		if err != nil {
			log.Println(err)
		}
		keyFile, err := writeContents(keyContents)
		if err != nil {
			log.Println(err)
		}
		conf.Key = keyFile

		// Retrieve Trust
		trustContents, err := testcerts.Asset("testcerts/client.trust.pem")
		if err != nil {
			log.Println(err)
		}
		trustFile, err := writeContents(trustContents)
		if err != nil {
			log.Println(err)
		}
		conf.Trust = trustFile

		// Set remaining defaults
		conf.SkipVerify = true
		conf.Remote = fmt.Sprintf("https://%s:%s/services/object-drive/1.0", config.DockerVM, config.Port)

	} else {
		// Parse the conf-file
		yamlFile, err := ioutil.ReadFile(confFile)
		if err != nil {
			return conf, err
		}

		err = yaml.Unmarshal(yamlFile, &conf)
		if err != nil {
			return conf, err
		}
	}

	return conf, nil
}

// writeContents dumps a byte array to a temporary file for use in
// functions that only accept a named file.  The location of the tmp file
// is the default temporary location for the OS, and the file will have a random
// string prefaced with "odrive-cli-temp".
func writeContents(content []byte) (string, error) {
	tmpfile, err := ioutil.TempFile("", "odrive-cli-temp")
	if err != nil {
		return "", nil
	}

	if _, err := tmpfile.Write(content); err != nil {
		return "", nil
	}
	if err := tmpfile.Close(); err != nil {
		return "", nil
	}

	return tmpfile.Name(), nil

}

// prettyPrint outputs an interface as either a
// JSON or YAML string.
func prettyPrint(v interface{}, format string) {
	var b []byte
	var err error

	if format == "json" {
		b, err = json.MarshalIndent(v, "", "    ")
		if err != nil {
			fmt.Println("JSON  error:", err)
		}
	} else if format == "yaml" {
		b, err = yaml.Marshal(v)
		if err != nil {
			fmt.Println("YAML  error:", err)
		}
	}

	fmt.Println(string(b))
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"decipher.com/object-drive-server/client"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
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

	testerFlag := cli.StringFlag{
		Name:  "tester",
		Value: "10",
		Usage: "tester credentials to use for connection",
	}

	jsonFlag := cli.BoolFlag{
		Name:  "json",
		Usage: "print all responses as formatted JSON",
	}

	yamlFlag := cli.BoolFlag{
		Name:  "yaml",
		Usage: "print all responses as formatted YAML",
	}

	queueFlag := cli.IntFlag{
		Name:  "queue",
		Value: 1000,
		Usage: "queue size in threaded upload",
	}

	threadFlag := cli.IntFlag{
		Name:  "threads",
		Value: 64,
		Usage: "number of threads to use in upload",
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
					Remote:     "https://host:port/path/to/service",
				}

				fmt.Println("# Example configuration file for odrive")
				prettyPrint(conf, "yaml")

				return nil
			},
		},
		{
			Name:  "test-connection",
			Usage: "establish connection to odrive and check for errors",
			Flags: []cli.Flag{confFlag, testerFlag},
			Action: func(clictx *cli.Context) error {

				conf, err := gatherConf(clictx.String("conf"), clictx.String("tester"))
				if err != nil {
					log.Println(err)
					return err
				}

				c, err := client.NewClient(conf)
				if err != nil {
					log.Println("could not establish connection", err)
					return err
				}

				log.Println("connection established successfully", c)
				return nil
			},
		},
		{
			Name:  "upload",
			Usage: "upload file to odrive",
			Flags: []cli.Flag{confFlag, jsonFlag, yamlFlag, testerFlag},
			Action: func(clictx *cli.Context) error {
				tester, err := parseTesterString(clictx.String("tester"))
				if err != nil {
					return err
				}

				conf, err := gatherConf(clictx.String("conf"), clictx.String("tester"))
				if err != nil {
					log.Println(err)
					return err
				}

				c, err := client.NewClient(conf)
				if err != nil {
					log.Println("could not establish connection", err)
					return err
				}

				var permissions = protocol.Permission{
					Read: protocol.PermissionCapability{
						AllowedResources: []string{fmt.Sprintf("user/cn=test tester%s,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester%s", tester, tester)},
					}}

				for _, fileName := range clictx.Args() {
					if !(clictx.Bool("json") || clictx.Bool("yaml")) {
						fmt.Printf("uploading %s...", fileName)
					}

					var obj = protocol.CreateObjectRequest{
						TypeName:   "File",
						Name:       fileName,
						RawAcm:     `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`,
						Permission: permissions,
						OwnedBy:    fmt.Sprintf("user/cn=test tester%s,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", tester),
					}

					fReader, err := os.Open(fileName)
					if err != nil {
						log.Println(err)
					}
					newObj, err := c.CreateObject(obj, fReader)
					if err != nil {
						log.Println("create error: ", err)
					}

					if clictx.Bool("json") {
						prettyPrint(newObj, "json")
					} else if clictx.Bool("yaml") {
						prettyPrint(newObj, "yaml")
					} else {
						fmt.Println("done")
					}

				}

				return nil
			},
		},
		{
			Name:  "test-fill",
			Usage: "upload a sample of random files and directories to the server",
			Flags: []cli.Flag{confFlag, jsonFlag, yamlFlag, testerFlag, queueFlag, threadFlag},
			Action: func(clictx *cli.Context) error {

				rand.Seed(time.Now().Unix())
				// Default to 10 items, parsing the first numerical argument if supplied
				// by the user.
				nFiles := 10
				if len(clictx.Args()) > 0 {
					nArg, err := strconv.Atoi(clictx.Args()[0])
					if err != nil {
						fmt.Println("argument counldn't parse to int:", clictx.Args()[0])
						return err
					}
					nFiles = nArg
				}

				newUserFillFunc := func() error {
					impersonation := false
					if len(clictx.Args()) > 1 && clictx.Args()[1] == "impersonation" {
						impersonation = true
					}
					var conf client.Config
					tester, err := parseTesterString(clictx.String("tester"))
					if err != nil {
						return err
					}
					username := fmt.Sprintf("test tester%s", tester)
					userdn := fmt.Sprintf("cn=%s,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", username)
					resource := fmt.Sprintf("user/%s/%s", userdn, username)
					ownedBy := fmt.Sprintf("user/%s", userdn)
					if impersonation {
						conf.Cert = "../../defaultcerts/server/server.cert.pem"
						conf.Key = "../../defaultcerts/server/server.key.pem"
						conf.Trust = "../../defaultcerts/server/server.trust.pem"
						conf.Impersonation = userdn
						//conf, err = gatherConfRaw(conf, clictx.String("conf"), cert, key, trust)
						conf.SkipVerify = true
						conf.Remote = fmt.Sprintf("https://proxier:%s/services/object-drive/1.0", config.Port)
						id := rand.Int31() % 5000
						username = fmt.Sprintf("usey%d mcuser%d", id, id)
						userdn = fmt.Sprintf("cn=%s,ou=aaa,o=u.s. government,c=us", username)
						resource = fmt.Sprintf("user/%s/%s", userdn, username)
						ownedBy = ""
					} else {
						conf, err = gatherConf(clictx.String("conf"), clictx.String("tester"))
						if err != nil {
							log.Println(err)
							return err
						}
					}
					c, err := client.NewClient(conf)
					if err != nil {
						log.Println("could not establish connection", err)
						return err
					}

					var permissions = protocol.Permission{
						Read: protocol.PermissionCapability{
							AllowedResources: []string{resource},
						},
					}
					fReader := randomFile()
					fakePath := randomPath()

					fullName := path.Join(fakePath, fReader.Name())

					var obj = protocol.CreateObjectRequest{
						TypeName:          "File",
						Name:              fullName,
						NamePathDelimiter: "/",
						RawAcm:            `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`,
						Permission:        permissions,
						OwnedBy:           ownedBy,
					}

					newObj, err := c.CreateObject(obj, fReader)
					if err != nil {
						log.Println("error on create: ", err)
						return err
					}
					fReader.Close()

					if clictx.Bool("json") {
						prettyPrint(newObj, "json")
					} else if clictx.Bool("yaml") {
						prettyPrint(newObj, "yaml")
					} else {
						fmt.Printf("uploaded: %s\n", fReader.Name())
					}

					os.RemoveAll(fReader.Name())
					return nil
				}

				// Fill a queue with tasks (could be much larger than the tasks queue)
				tasks := make(chan bool, clictx.Int("queue"))
				go func() {
					for i := 0; i < nFiles; i++ {
						tasks <- true
					}
					close(tasks)
				}()

				// Spawn nThreads to deal with nFiles tasks
				nThreads := clictx.Int("threads")
				wg := &sync.WaitGroup{}
				wg.Add(nThreads)
				for i := 0; i < nThreads; i++ {
					go func() {
						defer wg.Done()
						for _ = range tasks {
							newUserFillFunc()
						}
					}()
				}
				wg.Wait()
				return nil
			},
		},
	}

	// Global flags. Used when no "command" passed. Must be repeated above for commands.
	app.Flags = []cli.Flag{
		confFlag,
		jsonFlag,
		yamlFlag,
		testerFlag,
		queueFlag,
		threadFlag,
	}

	// There is no "default" command.  Print help and exit.
	app.Action = func(clictx *cli.Context) error {
		fmt.Printf("Must specify command. Run `%s help` for info\n", app.Name)
		return nil
	}

	app.Run(os.Args)
}

func gatherConf(confFile string, testerN string) (client.Config, error) {
	conf := client.Config{}
	i, err := strconv.Atoi(testerN)
	if err != nil {
		log.Println(err)
	}
	if i == 10 {
		i = 0
	}
	testerString := strconv.Itoa(i)
	cert := fmt.Sprintf("testcerts/test_%s.cert.pem", testerString)
	key := fmt.Sprintf("testcerts/test_%s.key.pem", testerString)
	trust := "testcerts/client.trust.pem"
	return gatherConfRaw(conf, confFile, cert, key, trust)
}

// gatherConf prepares the Config object necessary to perform actions in odrive.
// Calling gatherConf with a blank string invokes hard-coded default values and
// certificates, while calling with a named YAML file will load the given values.
// If a YAML file is specified, ALL values must be set.
func gatherConfRaw(conf client.Config, confFile string, cert, key, trust string) (client.Config, error) {
	// Retrieve Cert
	certContent, err := testcerts.Asset(cert)
	if err != nil {
		log.Println(err)
	}
	certFile, err := writeContents(certContent)
	if err != nil {
		log.Println(err)
	}
	conf.Cert = certFile

	// Retrieve Key
	keyContents, err := testcerts.Asset(key)
	if err != nil {
		log.Println(err)
	}
	keyFile, err := writeContents(keyContents)
	if err != nil {
		log.Println(err)
	}
	conf.Key = keyFile

	// Retrieve Trust
	trustContents, err := testcerts.Asset(trust)
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
	conf.Remote = fmt.Sprintf("https://proxier:%s/services/object-drive/1.0", config.Port)

	// Override supplied values
	if confFile != "" {
		log.Println("overriding defaults from ", confFile)
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

// randomPath creates a random string representing a valid path of directories.
func randomPath() string {
	randomName := func(name string) string {
		s, _ := util.NewGUID()
		return name + s
	}

	baseDir := "./"
	depth := rand.Intn(4)

	for i := 0; i < depth; i++ {
		baseDir = baseDir + randomName("child") + "/"
	}

	return baseDir
}

// randomFile opens a randomely named local file and appends a random
// body of characters into it.
func randomFile() *os.File {
	newFile, err := ioutil.TempFile("./", "testFile_")
	if err != nil {
		fmt.Println("Error writing temporary file", err)
	}

	body := randBody(rand.Intn(50))
	newFile.WriteString(body)

	return newFile
}

// randBody creates a random string of length n.
func randBody(n int) string {
	var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// parseTesterString translates a numerical string into the needed
// value to use as testerXX in sending and recieving data from odrive.
func parseTesterString(tester string) (string, error) {
	i, err := strconv.Atoi(tester)
	if err != nil {
		return "", err
	}
	testerString := strconv.Itoa(i)

	if i < 10 {
		testerString = "0" + testerString
	}

	return testerString, nil

}

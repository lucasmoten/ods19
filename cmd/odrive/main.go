package main

import (
	"fmt"
	"os"
	"path/filepath"

	"decipher.com/object-drive-server/ciphertext"

	"decipher.com/object-drive-server/amazon"

	"github.com/uber-go/zap"
	"github.com/urfave/cli"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/server"

	samuelzk "github.com/samuel/go-zookeeper/zk"
)

// Build and Commit are set at build time with -ldflags
var (
	Build  string
	Commit string
)

// Services that require network
const (
	S3Service        = "s3"
	AACService       = "aac"
	DatabaseService  = "db"
	ZookeeperService = "zk"
)

type emptyLogger struct{}

func (emptyLogger) Printf(format string, a ...interface{}) {
	//log.Printf(format, a...)
}

func main() {
	samuelzk.DefaultLogger = emptyLogger{}

	cliParser := cli.NewApp()
	cliParser.Name = "odrive"
	cliParser.Usage = "object-drive-server binary"
	cliParser.Version = fmt.Sprintf("1.0 - Build Number %s %s", Build, Commit)

	cliParser.Commands = []cli.Command{
		{
			Name:  "env",
			Usage: "Print all environment variables",
			Action: func(ctx *cli.Context) error {
				config.PrintODEnvironment()
				return nil
			},
		},
		{
			Name:  "makeScript",
			Usage: "Generate a startup script. Pipe output to a file.",
			Action: func(ctx *cli.Context) error {
				config.GenerateStartScript()
				return nil
			},
		},
		{
			Name:  "makeEnvScript",
			Usage: "List required env vars in script. Suitable for \"source\". Pipe output to a file.",
			Action: func(ctx *cli.Context) error {
				config.GenerateSourceEnvScript()
				return nil
			},
		},
		{
			Name:   "test",
			Usage:  "Run network diagnostic test against a service dependency. Values: s3, aac, db, zk",
			Action: serviceTest,
		},
	}

	var defaultCiphers cli.StringSlice
	defaultCiphers.Set("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")

	cliParser.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "addCipher",
			Usage: "A Go ciphersuite for TLS configuration. Can be specified multiple times. See: https://golang.org/src/crypto/tls/cipher_suites.go",
			Value: &defaultCiphers,
		},
		cli.BoolTFlag{
			Name:  "useTLS",
			Usage: "Serve content over TLS. Defaults to true.",
		},
		cli.StringSliceFlag{
			Name:  "whitelist",
			Usage: "Whitelisted DNs for impersonation",
		},
		cli.StringFlag{
			Name:  "conf",
			Usage: "Path to yaml configuration file.",
			Value: "odrive.yml",
		},
		cli.StringFlag{
			Name:  "staticRoot",
			Usage: "Path to static files. Defaults to libs/server/static",
			Value: filepath.Join("..", "..", "server", "static"),
		},
		cli.StringFlag{
			Name:  "templateDir",
			Usage: "Path to template files. Defaults to libs/server/static/templates",
			Value: filepath.Join("..", "..", "server", "static", "templates"),
		},
		cli.StringFlag{
			Name:  "tlsMinimumVersion",
			Usage: "Minimum Version of TLS to support (defaults to 1.2, valid values are 1.0, 1.1)",
			Value: "1.2",
		},
	}

	cliParser.Action = func(c *cli.Context) error {

		opts := config.NewCommandLineOpts(c)
		conf := config.NewAppConfiguration(opts)

		config.RootLogger.Info("configuration-settings", zap.String("confPath", opts.Conf),
			zap.String("staticRoot", opts.StaticRootPath),
			zap.String("templateDir", opts.TemplateDir),
			zap.String("tlsMinimumVersion", opts.TLSMinimumVersion))

		err := server.Start(conf)
		return err
	}

	cliParser.Run(os.Args)
}

func serviceTest(ctx *cli.Context) error {
	service := ctx.Args().First()
	switch service {
	case S3Service:
		s3Config := config.NewS3Config()
		if !ciphertext.TestS3Connection(amazon.NewAWSSession(s3Config.AWSConfig, config.RootLogger)) {
			fmt.Println("ERROR: Cannot access S3 bucket.")
			os.Exit(1)
		} else {
			fmt.Println("SUCCESS: Can read and write bucket referenced by OD_AWS_S3_BUCKET")
			os.Exit(0)
		}
	case AACService:
		fmt.Println("Not implemented for service:", service)
	case DatabaseService:
		fmt.Println("Not implemented for service:", service)
	case ZookeeperService:
		fmt.Println("Not implemented for service:", service)
	default:
		fmt.Println("Unknown service. Please run `odrive help`")
	}
	return nil
}

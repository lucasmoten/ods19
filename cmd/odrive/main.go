package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/Shopify/sarama"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/urfave/cli"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/amazon"
	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/server"
	"bitbucket.di2e.net/dime/object-drive-server/services/kafka"
)

// Version metadata should be set at build time with -ldflags.
var (
	Build   string
	Commit  string
	Version string
)

func main() {
	zk.DefaultLogger = emptyLogger{}

	cliParser := cli.NewApp()
	cliParser.Name = "odrive"
	cliParser.Usage = "object-drive-server binary"
	if len(Version) > 0 && len(Build) > 0 {
		cliParser.Version = fmt.Sprintf("%s build %s (%s)", Version, Build, Commit)
	}

	var defaultCiphers cli.StringSlice
	defaultCiphers.Set("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")

	globalFlags := []cli.Flag{
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

	cliParser.Commands = []cli.Command{
		{
			Name:  "schemaversion",
			Usage: "Expected DB Schema version in use.",
			Action: func(ctx *cli.Context) error {
				fmt.Printf("Schema Version Needed: %s\n", dao.SchemaVersion)
				return nil
			},
		},
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
			Flags:  globalFlags,
			Action: serviceTest,
		},
		{
			Name:  "isfips",
			Usage: "Report FIPS 140-2 compliance",
			Action: func(ctx *cli.Context) error {
				fmt.Println("FIPS 140-2 compliance check for BoringCrypto module")
				fmt.Println()
				fmt.Printf("Built with Go runtime.version: %s\n", runtime.Version())

				rbc := regexp.MustCompile(`(?P<base_go_version>go[\d\.]*)(?P<boringcrypto_enabled>b)(?P<boringcrypto_update_version>\d*)`)
				mbc := rbc.FindStringSubmatch(runtime.Version())
				if len(mbc) > 0 {
					for i, n := range rbc.SubexpNames() {
						if i > 0 {
							fmt.Printf("\t%s = %s\n", n, mbc[i])
						}
					}
					fmt.Println("The version of Go used to compile this binary uses BoringCrypto")
				} else {
					fmt.Println("The Go runtime.version doesn't appear to include BoringCrypto")
				}
				fmt.Println()
				fmt.Println("If Go is available, you may also check whether the binary is using symbols from the module ")
				fmt.Println("  go tool nm odrive | grep \"crypto/internal/boring._cgo\" ")
				return nil
			},
		},
	}

	cliParser.Flags = globalFlags

	cliParser.Action = func(c *cli.Context) error {
		opts := config.NewCommandLineOpts(c)
		conf := config.NewAppConfiguration(opts)
		config.RootLogger.Info("configuration-settings", zap.String("confPath", opts.Conf),
			zap.String("staticRoot", opts.StaticRootPath),
			zap.String("templateDir", opts.TemplateDir),
			zap.String("tlsMinimumVersion", opts.TLSMinimumVersion))

		conf.ServerSettings.Version = cliParser.Version

		for _, v := range conf.ServerSettings.ACLImpersonationWhitelist {
			config.RootLogger.Info("permitted to impersonate", zap.String("whitelisted dn", v))
		}

		rbc := regexp.MustCompile(`(?P<base_go_version>go[\d\.]*)(?P<boringcrypto_enabled>b)(?P<boringcrypto_update_version>\d*)`)
		mbc := rbc.FindStringSubmatch(runtime.Version())
		if len(mbc) > 0 {
			config.RootLogger.Info("boring-crypto", zap.String("update", mbc[3]), zap.String("runtime.Version", runtime.Version()))
		}

		err := server.Start(conf)
		return err
	}

	cliParser.Run(os.Args)
}

// Services available for testing.
const (
	S3Service        = "s3"
	AACService       = "aac"
	DatabaseService  = "db"
	ZookeeperService = "zk"
	KafkaService     = "kafka"
)

type emptyLogger struct{}

func (emptyLogger) Printf(format string, a ...interface{}) {}

func serviceTest(ctx *cli.Context) error {
	opts := config.NewCommandLineOpts(ctx)
	conf := config.NewAppConfiguration(opts)

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
	case KafkaService:

		fmt.Println("Printing config")
		fmt.Printf("=> Kafka-specific zk (%s): %v\n",
			config.OD_EVENT_ZK_ADDRS, conf.EventQueue.ZKAddrs)
		fmt.Printf("=> Kafka direct-connect vals (%s): %v\n",
			config.OD_EVENT_KAFKA_ADDRS, conf.EventQueue.KafkaAddrs)
		if len(conf.EventQueue.KafkaAddrs) > 0 {
			fmt.Println("=> OD_EVENT_KAFKA_ADDRS is not empty; Kafka will be connected to directly")
		} else if len(conf.EventQueue.ZKAddrs) > 0 {
			fmt.Println("=> OD_EVENT_ZK_ADDRS is not empty; Kafka will be discovered from ZK")
		} else {
			fmt.Println("=> OD_EVENT_KAFKA_ADDRS and OD_EVENT_ZK_ADDRS are empty; No events will be published.")
		}
		fmt.Println("Testing Kafka-related services...")

		kafkaInfo := func(c sarama.Client) error {
			defer c.Close()
			fmt.Println("Connection to kafka successful")

			topics, err := c.Topics()
			if err != nil {
				return fmt.Errorf("could not get topics: %v", err)
			}
			var found bool
			for _, t := range topics {
				if t == conf.EventQueue.Topic {
					found = true
				}
			}
			if !found {
				fmt.Printf("Topic %s was not found\n", conf.EventQueue.Topic)
			} else {
				fmt.Printf("Topic %s found\n", conf.EventQueue.Topic)
				n, err := c.WritablePartitions(conf.EventQueue.Topic)
				if err != nil {
					return fmt.Errorf("error getting writable partitions for %s topic: %v", conf.EventQueue.Topic, err)
				}
				fmt.Printf("Found %v writable partition(s) for %s topic\n", len(n), conf.EventQueue.Topic)
			}
			return nil
		}

		testZK := func() error {
			if len(conf.EventQueue.ZKAddrs) > 0 {
				fmt.Println("=> Attempting to connect to ZK cluster for Kafka discovery")
				timeout := 5 * time.Second
				conn, _, err := zk.Connect(conf.EventQueue.ZKAddrs, time.Duration(timeout))
				if err != nil {
					return err
				}
				fmt.Println("=> Connected to zk; searching for kafka brokers at default path /brokers/ids")
				brokers := kafka.BrokersFromZKPath(conn, "/brokers/ids")
				if len(brokers) < 1 {
					return errors.New("no broker data found at Kafka path")
				}
				client, err := sarama.NewClient(brokers, nil)
				if err != nil {
					return err
				}
				return kafkaInfo(client)
			}
			return nil
		}

		testKafkaDirect := func() error {
			fmt.Println("=> Attempting direct connect to Kafka cluster")

			if len(conf.EventQueue.KafkaAddrs) > 0 {
				client, err := sarama.NewClient(conf.EventQueue.KafkaAddrs, nil)
				if err != nil {
					return err
				}
				return kafkaInfo(client)
			}
			return nil
		}
		err := testKafkaDirect()
		if err != nil {
			fmt.Println("Direct connect to Kafka had errors:", err)
		}
		err = testZK()
		if err != nil {
			fmt.Println("Kafka discovery from ZK had errors:", err)
		}

	default:
		fmt.Println("Unknown service. Please run `odrive help`")
	}
	return nil
}

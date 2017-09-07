package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Shopify/sarama"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/uber-go/zap"
	"github.com/urfave/cli"

	"decipher.com/object-drive-server/amazon"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/services/kafka"
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
	cliParser.Version = fmt.Sprintf("%s build :%s", Version, Build)

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
	}

	cliParser.Flags = globalFlags

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
				if t == "odrive-event" {
					found = true
				}
			}
			if !found {
				fmt.Println("Topic odrive-event was not found")
			} else {
				fmt.Println("Topic odrive-event found")
				n, err := c.WritablePartitions("odrive-event")
				if err != nil {
					return fmt.Errorf("error getting writable partitions for odrive-event topic: %v", err)
				}
				fmt.Printf("Found %v writable partition(s) for odrive-event topic\n", len(n))
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

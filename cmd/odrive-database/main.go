package main

import (
	"os"

	configx "decipher.com/object-drive-server/configx"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "odrive-database"
	app.Usage = "odrive database manager for setup and migrations"
	app.Version = "1.0"

	app.Commands = []cli.Command{
		{
			Name:  "status",
			Usage: "Print status for configured database",
			Action: func(ctx *cli.Context) error {
				configx.PrintODEnvironment()
				return nil
			},
		},
	}

	var defaultCiphers cli.StringSlice
	defaultCiphers.Set("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "conf",
			Usage: "Path to yaml configuration file.",
			Value: "odrive.yml",
		},
	}

	app.Action = func(c *cli.Context) error {

		run(c)
		return nil
	}

	app.Run(os.Args)

}

func run(c *cli.Context) {

}

func status() {
}

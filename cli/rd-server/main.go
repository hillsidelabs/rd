package main

import (
	"log"
	"os"

	"github.com/hillside-labs/rd/web"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "rd-server",
		Usage: "Start the web server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   "8080",
				Usage:   "Port to run the server on",
			},
		},
		Action: func(c *cli.Context) error {
			port := c.String("port")
			server := web.NewServer(port)
			return server.Start()
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}


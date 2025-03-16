package main

import (
	"github.com/hillside-labs/rd/infra"
	"github.com/urfave/cli/v2"
)

var dockerStatusCmd = []string{"docker", "compose", "ps"}

func runDockerCmd(targets []infra.Host, cmd []string, arg string) {
	if arg != "" {
		cmd = append(cmd, arg)
	}

	for _, t := range targets {
		ExecuteCmd(t, cmd...)
	}
}

func runDockerStatus(targets []infra.Host) {
	for _, t := range targets {
		ExecuteCmd(t, "docker", "ps")
	}
}

func addDockerCmds(cmd []*cli.Command) []*cli.Command {
	cmd = append(cmd, []*cli.Command{
		{
			Name:  "start",
			Usage: "Starts the compose processes.",
			Flags: []cli.Flag{nameFlag, ipFlag, privateFlag},
			Action: func(c *cli.Context) error {
				targets, err := GetTargetsWithFlags(c)
				if err != nil {
					return err
				}

				args := []string{
					"docker", "compose", "up", "-d",
				}
				if c.Args().First() != "" {
					args = append(args, c.Args().First())
				}

				for _, t := range targets {
					ExecuteCmd(t, args...)
					ExecuteCmd(t, dockerStatusCmd...)
				}

				return nil

			},
		},
		{
			Name:    "update",
			Aliases: []string{"pull"},
			Usage:   "Pull the containers.",
			Flags:   []cli.Flag{nameFlag, ipFlag, privateFlag},
			Action: func(c *cli.Context) error {
				targets, err := GetTargetsWithFlags(c)
				if err != nil {
					return err
				}

				runDockerCmd(targets, []string{"docker", "compose", "pull"}, c.Args().First())

				return nil
			},
		},

		{
			Name:  "restart",
			Usage: "Restart the compose processes.",
			Flags: []cli.Flag{nameFlag, ipFlag, privateFlag},
			Action: func(c *cli.Context) error {
				targets, err := GetTargetsWithFlags(c)
				if err != nil {
					return err
				}

				runDockerCmd(targets, []string{"docker", "compose", "restart"}, c.Args().First())
				runDockerStatus(targets)

				return nil
			},
		},
		{
			Name:  "reboot",
			Usage: "Stop, update, and start the compose containers.",
			Flags: []cli.Flag{nameFlag, ipFlag, privateFlag},
			Action: func(c *cli.Context) error {
				targets, err := GetTargetsWithFlags(c)
				if err != nil {
					return err
				}

				runDockerCmd(targets, []string{"docker", "compose", "stop"}, c.Args().First())
				runDockerCmd(targets, []string{"docker", "compose", "up", "-d"}, c.Args().First())
				runDockerStatus(targets)
				return nil
			},
		},
		{
			Name:  "logs",
			Usage: "Tail the docker logs.",
			Flags: []cli.Flag{nameFlag, ipFlag, privateFlag},
			Action: func(c *cli.Context) error {
				targets, err := GetTargetsWithFlags(c)
				if err != nil {
					return err
				}
				runDockerCmd(targets, []string{"docker", "compose", "logs", "-f"}, c.Args().First())

				return nil
			},
		},
		{
			Name:    "ps",
			Aliases: []string{"status"},
			Usage:   "Docker compose ps",
			Flags:   []cli.Flag{nameFlag, ipFlag, privateFlag},
			Action: func(c *cli.Context) error {
				targets, err := GetTargetsWithFlags(c)
				if err != nil {
					return err
				}
				runDockerStatus(targets)
				return nil
			},
		},
	}...)

	return cmd
}

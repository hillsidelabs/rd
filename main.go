package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/hillside-labs/rd/infra"
	"github.com/hillside-labs/rd/infra/amazon"
	"github.com/ionrock/procs"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/urfave/cli/v2"
)

func GetTargetsWithFlags(c *cli.Context) ([]infra.Host, error) {
	catalog, err := infra.NewCatalog()
	if err != nil {
		return nil, err
	}

	return catalog.GetTargets(
		c.String("name"),
		c.String("ip"),
		c.String("private"),
	)
}

func rdUser() string {
	user := os.Getenv("RD_USER")
	if user == "" {
		user = "root"
	}

	return user
}

func NewSSHCmd(host infra.Host, args ...string) *exec.Cmd {
	user := rdUser()
	conn := fmt.Sprintf("%s@%s", user, host.IP)
	command := exec.Command("ssh", conn)
	command.Args = append(command.Args, args...)

	return command
}

func ExecuteCmd(host infra.Host, args ...string) {
	command := NewSSHCmd(host, args...)
	p := procs.Process{Cmds: []*exec.Cmd{command}}
	p.OutputHandler = func(line string) string {
		fmt.Printf("%s\t| %s\n", host.Name, line)
		return line
	}

	p.ErrHandler = p.OutputHandler

	fmt.Printf("%s\t| Running: %s\n", host.Name, command.String())
	p.Run()
}

func SyncFiles(host infra.Host, src string, recursive bool) {
	user := rdUser()
	dest := fmt.Sprintf("%s@%s:.", user, host.IP)
	command := exec.Command("scp")
	if recursive {
		command.Args = append(command.Args, "-r")
	}
	command.Args = append(command.Args, src, dest)
	p := procs.Process{Cmds: []*exec.Cmd{command}}
	p.OutputHandler = func(line string) string {
		fmt.Printf("%s | %s\n", host.Name, line)
		return line
	}

	fmt.Printf("%s\t| Running: %s\n", host.Name, command.String())
	p.Run()
}

var nameFlag = &cli.StringFlag{
	Name:    "name",
	Usage:   "filter hosts by name",
	Aliases: []string{"n"},
}
var ipFlag = &cli.StringFlag{
	Name:    "ip",
	Usage:   "filter hosts by public ip",
	Aliases: []string{"i"},
}
var privateFlag = &cli.StringFlag{
	Name:    "private",
	Usage:   "filter hosts by public ip",
	Aliases: []string{"p"},
}

func main() {

	app := cli.App{
		Name:  "rd",
		Usage: "rd stands for remote docker",
		Commands: []*cli.Command{
			{
				Name:  "bootstrap",
				Usage: "Setup a host with docker for rd",
				Flags: []cli.Flag{
					nameFlag, ipFlag, privateFlag},
				Action: func(c *cli.Context) error {
					fname, err := BoostrapScript()
					if err != nil {
						return err
					}

					defer os.Remove(fname)

					targets, err := GetTargetsWithFlags(c)

					if err != nil {
						return err
					}

					for _, t := range targets {
						SyncFiles(t, fname, false)
						ExecuteCmd(t, "chmod", "+x", "bootstrap.sh")

						// TODO: Might need sudo here...
						ExecuteCmd(t, "./bootstrap.sh")
					}

					return nil
				},
			},
			{
				Name:  "sync",
				Usage: "Copy files to the remote host. If no file arguments are given, it copies the docker-compose.yml.",
				Flags: []cli.Flag{
					nameFlag, ipFlag, privateFlag,
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "Sync recursively like scp -r",
					},
				},
				Action: func(c *cli.Context) error {
					targets, err := GetTargetsWithFlags(c)

					if err != nil {
						return err
					}

					fn := c.Args().First()
					if fn == "" {
						fn = "docker-compose.yml"
					}

					for _, t := range targets {
						SyncFiles(t, fn, c.Bool("recursive"))
					}
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "Run a command on the remote host.",
				Flags: []cli.Flag{nameFlag, ipFlag, privateFlag},
				Action: func(c *cli.Context) error {
					targets, err := GetTargetsWithFlags(c)
					if err != nil {
						return err
					}
					for _, t := range targets {
						fmt.Println(t)
					}

					for _, t := range targets {
						ExecuteCmd(t, c.Args().Slice()...)
					}

					return nil
				},
			},
			{
				Name:  "config",
				Usage: "Get configuration data from DO spaces.",
				Action: func(c *cli.Context) error {

					endpoint := "sfo3.digitaloceanspaces.com"
					useSSL := true
					spacesKey := os.Getenv("AWS_ACCESS_KEY_ID")
					spacesSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")

					bucket := c.Args().First()
					key := c.Args().Get(1)

					if bucket == "" || key == "" {
						return fmt.Errorf("Missing bucket: %s or key: %s", bucket, key)
					}

					minioClient, err := minio.New(endpoint, &minio.Options{
						Creds:  credentials.NewStaticV4(spacesKey, spacesSecret, ""),
						Secure: useSSL,
					})

					if err != nil {
						return err
					}

					obj, err := minioClient.GetObject(context.Background(), bucket, key, minio.GetObjectOptions{})
					if err != nil {
						return err
					}
					defer obj.Close()

					content, err := io.ReadAll(obj)
					if err != nil {
						return err
					}
					fmt.Println(string(content))

					return err

				},
			},
			{
				Name:  "hosts",
				Usage: "List the hosts.",
				Flags: []cli.Flag{nameFlag, ipFlag, privateFlag},
				Action: func(c *cli.Context) error {
					targets, err := GetTargetsWithFlags(c)
					if err != nil {
						return err
					}

					for _, t := range targets {
						fmt.Println(t.Name, t.IP, t.PrivateIP)
					}

					return nil
				},
			},
			{
				Name:  "deploy",
				Usage: "Sync, pull, and restart the compose containers.",
				Flags: []cli.Flag{nameFlag, ipFlag, privateFlag,
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "Sync recursively like scp -r",
					}},
				Action: func(c *cli.Context) error {
					targets, err := GetTargetsWithFlags(c)
					if err != nil {
						return err
					}

					fn := c.Args().First()
					if fn == "" {
						fn = "docker-compose.yml"
					}

					for _, t := range targets {
						SyncFiles(t, fn, c.Bool("recursive"))
					}

					runDockerCmd(targets, []string{"docker", "compose", "pull"}, "")
					runDockerCmd(targets, []string{"docker", "compose", "stop"}, "")
					runDockerCmd(targets, []string{"docker", "compose", "up", "-d"}, "")
					runDockerStatus(targets)

					return nil
				},
			},
			{
				Name:  "infra",
				Usage: "Manage infrastructure",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a new VM instance",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "Name of the VM instance",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "image-id",
								Usage: "AWS AMI ID",
								Value: "ami-02fe0558ef9c00c8c", // Ubuntu 24.10 us-west-2
							},
							&cli.StringFlag{
								Name:  "type",
								Usage: "Instance type",
								Value: "t2.micro",
							},
							&cli.StringSliceFlag{
								Name:  "tag",
								Usage: "Tags to apply to the instance (can be specified multiple times)",
							},
							&cli.StringFlag{
								Name:  "region",
								Usage: "AWS region",
								Value: "us-west-2",
							},
						},
						Action: func(c *cli.Context) error {
							// Create AWS config with custom region
							config := amazon.NewAWSConfig()
							config.Region = c.String("region")

							// Create the VM
							vm := &amazon.VM{
								Config:       config,
								Name:         c.String("name"),
								ImageID:      c.String("image-id"),
								InstanceType: c.String("type"),
								Tags:         c.StringSlice("tag"),
								VPC:          c.String("name"), // Use name as VPC name
							}

							// Create the VM instance
							if err := vm.Create(); err != nil {
								return fmt.Errorf("failed to create VM: %v", err)
							}

							fmt.Printf("Successfully created VM '%s'\n", vm.Name)
							return nil
						},
					},
				},
			},
		},
	}

	app.Commands = addDockerCmds(app.Commands)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

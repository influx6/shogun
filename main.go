package main

//go:generate go generate ./...

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/minio/cli"
)

// Version defines the version number for the cli.
var Version = "0.1"

var helpTemplate = `NAME:
{{.Name}} - {{.Usage}}

VERSION:
{{.Version}}

DESCRIPTION:
{{.Description}}

USAGE:
{{.Name}} {{if .Flags}}[flags] {{end}}command{{if .Flags}}{{end}} [arguments...]

COMMANDS:
	{{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
	{{end}}{{if .Flags}}
FLAGS:
	{{range .Flags}}{{.}}
	{{end}}{{end}}

`

func main() {
	app := cli.NewApp()
	app.Name = "Shogun"
	app.Author = "Ewetumo Alexander"
	app.Email = "trinoxf@gmail.com"
	app.Usage = "shogun {{command}}"
	app.Flags = []cli.Flag{}
	app.Description = "Shogun: Become one with your katana functions"
	app.CustomAppHelpTemplate = helpTemplate
	app.Action = func(c *cli.Context) error {
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:   "version",
			Action: versionAction,
			Flags:  []cli.Flag{},
		},
	}

	app.RunAndExitOnError()
}

func versionAction(c *cli.Context) {
	fmt.Println(color.BlueString(fmt.Sprintf("shogun %s %s/%s", Version, runtime.GOOS, runtime.GOARCH)))
}

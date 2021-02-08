package main

import (
	cli "github.com/andreaskaris/sosreport-operator-cli/pkg/cli"
	client "github.com/andreaskaris/sosreport-operator-cli/pkg/client"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
)

func PrintError(err error) {
	log.Fatal(os.Stderr, err.Error())
	os.Exit(1)
}

func main() {
	commandLine, err := cli.NewCli()
	if err != nil {
		PrintError(err)
	}

	log.SetLevel(commandLine.LogLevel)
	log.SetFormatter(&log.TextFormatter{
            TimestampFormat : "2006-01-02 15:04:05",
            FullTimestamp:true,
        })

	log.Debug(fmt.Sprintf("Flags:\n%s", commandLine.PrintFlags()))

	c, err := client.NewClient()
	if err != nil {
		PrintError(err)
	}

	if err = c.CreateSosreport(commandLine); err != nil {
		PrintError(err)
	}
}

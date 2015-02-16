package main

import (
	"github.com/optionfactory/gdrive2slack/gdrive2slack"
	"os"
)

var version string

func main() {
	logger := gdrive2slack.NewLogger(os.Stdout, "", 0)
	logger.Info("gdrive2slack version:%s", version)
	if len(os.Args) != 2 {
		logger.Error("usage: %s <configuration_file>", os.Args[0])
		os.Exit(1)
	}

	configuration, err := gdrive2slack.LoadConfiguration(os.Args[1])
	if err != nil {
		logger.Error("cannot read configuration: %s", err)
		os.Exit(1)
	}
	env := gdrive2slack.NewEnvironment(version, configuration, logger)

	go gdrive2slack.EventLoop(env)
	gdrive2slack.ServeHttp(env)
}

package main

import (
	"github.com/optionfactory/gdrive2slack/gdrive2slack"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version string

func main() {
	logger := gdrive2slack.New(os.Stdout, "", 0)
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

	registerChannel := make(chan *gdrive2slack.SubscriptionAndAccessToken, 50)
	discardChannel := make(chan string, 50)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.Signal(0xf))

	client := &http.Client{
		Timeout: time.Duration(15) * time.Second,
	}

	go gdrive2slack.EventLoop(configuration, logger, client, registerChannel, discardChannel, signals, version)

	gdrive2slack.ServeHttp(client, registerChannel, configuration, version)
}

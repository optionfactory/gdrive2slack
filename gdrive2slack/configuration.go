package gdrive2slack

import (
	"encoding/json"
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/mailchimp"
	"github.com/optionfactory/gdrive2slack/slack"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Configuration struct {
	BindAddress      string                     `json:"bindAddress"`
	GoogleTrackingId string                     `json:"googleTrackingId"`
	Google           *google.OauthConfiguration `json:"google"`
	Slack            *slack.OauthConfiguration  `json:"slack"`
	Mailchimp        *mailchimp.Configuration   `json:"mailchimp"`
}

func LoadConfiguration(filename string) (*Configuration, error) {
	var self = new(Configuration)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(self)
	if err != nil {
		return nil, err
	}
	return self, nil
}

type Environment struct {
	Version         string
	Configuration   *Configuration
	Logger          *Logger
	HttpClient      *http.Client
	RegisterChannel chan *SubscriptionAndAccessToken
	DiscardChannel  chan string
	SignalsChannel  chan os.Signal
}

func NewEnvironment(version string, conf *Configuration, logger *Logger) *Environment {
	e := &Environment{
		Version:       version,
		Configuration: conf,
		Logger:        logger,
		HttpClient: &http.Client{
			Timeout: time.Duration(15) * time.Second,
		},
		RegisterChannel: make(chan *SubscriptionAndAccessToken, 50),
		DiscardChannel:  make(chan string, 50),
		SignalsChannel:  make(chan os.Signal, 1),
	}
	signal.Notify(e.SignalsChannel, syscall.SIGINT, syscall.Signal(0xf))
	return e
}

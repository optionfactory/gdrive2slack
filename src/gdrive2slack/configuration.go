package gdrive2slack

import (
	"encoding/json"
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/mailchimp"
	"github.com/optionfactory/gdrive2slack/slack"
	"os"
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

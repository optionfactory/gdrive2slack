package gdrive2slack

import (
	"encoding/json"
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	"os"
)

type UserState struct {
	Channel        string             `json:"channel"`
	Slack          *slack.OauthState  `json:"soauth"`
	Google         *google.OauthState `json:"goauth"`
	GoogleUserInfo *userinfo.UserInfo `json:"guser"`
	SlackUserInfo  *slack.UserInfo    `json:"suser"`
	Gdrive         *drive.State       `json:"-"`
}

func LoadUserStates(filename string) (map[string]*UserState, error) {
	var self = make(map[string]*UserState)
	file, err := os.Open(filename)
	if err != nil {
		return self, nil
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&self)
	if err != nil {
		return nil, err
	}
	for _, v := range self {
		v.Gdrive = drive.NewState()
	}
	return self, nil
}

func SaveUserStates(states map[string]*UserState, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(states)
}

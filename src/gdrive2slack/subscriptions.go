package gdrive2slack

import (
	"encoding/json"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	"os"
)

type Subscription struct {
	Channel            string             `json:"channel"`
	SlackAccessToken   string             `json:"slack_access_token"`
	GoogleRefreshToken string             `json:"google_refresh_token"`
	GoogleUserInfo     *userinfo.UserInfo `json:"guser"`
	SlackUserInfo      *slack.UserInfo    `json:"suser"`
}

type UserState struct {
	Gdrive            *drive.State
	GoogleAccessToken string
}

type SubscriptionAndAccessToken struct {
	Subscription      *Subscription
	GoogleAccessToken string
}

func LoadSubscriptions(filename string) (map[string]*UserState, map[string]*Subscription, error) {
	var subscriptions = make(map[string]*Subscription)
	var states = make(map[string]*UserState)
	file, err := os.Open(filename)
	if err != nil {
		return states, subscriptions, nil
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&subscriptions)
	if err != nil {
		return nil, nil, err
	}
	for k := range subscriptions {
		states[k] = &UserState{
			Gdrive:            drive.NewState(),
			GoogleAccessToken: "",
		}
	}
	return states, subscriptions, nil
}

func SaveSubscriptions(states map[string]*Subscription, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(states)
}

func AddSubscription(userStates map[string]*UserState, subscriptions map[string]*Subscription, subscription *Subscription, googleAccessToken string) {
	subscriptions[subscription.GoogleUserInfo.Email] = subscription
	userStates[subscription.GoogleUserInfo.Email] = &UserState{
		Gdrive:            drive.NewState(),
		GoogleAccessToken: googleAccessToken,
	}
}

func RemoveSubscription(userStates map[string]*UserState, subscriptions map[string]*Subscription, email string) {
	delete(subscriptions, email)
	delete(userStates, email)
}

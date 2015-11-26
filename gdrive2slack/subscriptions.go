package gdrive2slack

import (
	"encoding/json"
	"fmt"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	"os"
	"time"
)

type Subscription struct {
	Channel                    string             `json:"channel"`
	SlackAccessToken           string             `json:"slack_access_token"`
	GoogleRefreshToken         string             `json:"google_refresh_token"`
	GoogleUserInfo             *userinfo.UserInfo `json:"guser"`
	SlackUserInfo              *slack.UserInfo    `json:"suser"`
	GoogleInterestingFolderIds []string           `json:"google_interesting_folder_ids"`
}

type UserState struct {
	Gdrive            *drive.State
	GoogleAccessToken string
	FailingSince      *time.Time
}

type SubscriptionAndAccessToken struct {
	Subscription      *Subscription
	GoogleAccessToken string
}

type Subscriptions struct {
	Source string
	Info   map[string]*Subscription
	States map[string]*UserState
}

func LoadSubscriptions(filename string) (*Subscriptions, error) {
	var subscriptions = &Subscriptions{
		Source: filename,
		Info:   make(map[string]*Subscription),
		States: make(map[string]*UserState),
	}
	file, err := os.Open(filename)
	if err != nil {
		return subscriptions, nil
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&subscriptions.Info)
	if err != nil {
		return nil, err
	}
	for k, sub := range subscriptions.Info {
		subscriptions.States[k] = &UserState{
			Gdrive:            drive.NewState(),
			GoogleAccessToken: "",
			FailingSince:      nil,
		}
		// handle migration from versions prior to folder filtering
		if sub.GoogleInterestingFolderIds == nil {
			sub.GoogleInterestingFolderIds = make([]string, 0)
		}
	}
	return subscriptions, nil
}

func (subscriptions *Subscriptions) save() error {
	s := func(filename string) error {
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		return json.NewEncoder(file).Encode(subscriptions.Info)
	}
	suffix := time.Now().Format("2006-01-02T15-04-05")
	err1 := s(subscriptions.Source)
	err2 := s(subscriptions.Source + "." + suffix)
	if err1 != nil {
		return err1
	}
	return err2
}

func (subscriptions *Subscriptions) Add(subscription *Subscription, googleAccessToken string) {
	subscriptions.Info[subscription.GoogleUserInfo.Email] = subscription
	subscriptions.States[subscription.GoogleUserInfo.Email] = &UserState{
		Gdrive:            drive.NewState(),
		GoogleAccessToken: googleAccessToken,
	}
	subscriptions.save()
}

func (subscriptions *Subscriptions) HandleFailure(email string) (*Subscription, string, bool) {
	s := subscriptions.Info[email]
	state := subscriptions.States[email]
	now := time.Now()
	threshold := now.Add(-24 * time.Hour)
	if state.FailingSince == nil {
		state.FailingSince = &now
		return s, "new_failure", false
	}
	if state.FailingSince.Before(threshold) {
		delete(subscriptions.States, email)
		delete(subscriptions.Info, email)
		subscriptions.save()
		return s, fmt.Sprintf("over_failure_threshold_since@%v", state.FailingSince.Unix()), true
	}
	return s, fmt.Sprintf("still_failing_since@%v", state.FailingSince.Unix()), false

}

func (subscriptions *Subscriptions) HandleSuccess(email string) {
	subscriptions.States[email].FailingSince = nil
}

func (subscriptions *Subscriptions) Contains(email string) bool {
	_, ok := subscriptions.Info[email]
	return ok
}

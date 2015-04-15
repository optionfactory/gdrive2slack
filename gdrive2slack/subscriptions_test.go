package gdrive2slack

import (
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	_ "log"
	"os"
	"testing"
)

func TestCanDeserializeANonexistentFile(t *testing.T) {
	subs, err := LoadSubscriptions("/tmp/not-a-real-file")
	if err != nil || len(subs.Info) != 0 {
		t.Fail()
	}
	_ = subs
}

func TestAddingToSubscriptionsUpdatesTheFile(t *testing.T) {
	subs, err := LoadSubscriptions("/tmp/not-a-real-file")
	subs.Source = "/tmp/temp-subs"
	subscription := &Subscription{
		Channel:            "channel",
		SlackAccessToken:   "slack-token",
		GoogleRefreshToken: "g-refresh-token",
		GoogleUserInfo:     &userinfo.UserInfo{},
		SlackUserInfo:      &slack.UserInfo{},
	}
	subs.Add(subscription, "a-fake-token")
	defer os.Remove(subs.Source)
	fi, err := os.Lstat(subs.Source)
	if err != nil || fi.Size() == 0 {
		t.Fail()
	}
}

func TestCorrectlyDeserializeSerializedSubscriptions(t *testing.T) {
	subs, _ := LoadSubscriptions("/tmp/not-a-real-file")
	subs.Source = "/tmp/temp-subs"
	subscription := &Subscription{
		Channel:            "channel",
		SlackAccessToken:   "slack-token",
		GoogleRefreshToken: "g-refresh-token",
		GoogleUserInfo:     &userinfo.UserInfo{},
		SlackUserInfo:      &slack.UserInfo{},
	}
	subs.Add(subscription, "a-fake-token")
	defer os.Remove(subs.Source)
	deserialized, _ := LoadSubscriptions(subs.Source)
	if len(subs.Info) != len(deserialized.Info) {
		t.Fail()
	}
}

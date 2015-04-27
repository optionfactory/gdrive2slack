package gdrive2slack

import (
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	_ "log"
	"os"
	"path/filepath"
	"testing"
)

func cleanup(t *testing.T, root string, pattern string) {
	filepath.Walk(root, func(path string, f os.FileInfo, err error) (e error) {
		if matches, _ := filepath.Match(pattern, f.Name()); matches {
			t.Log("matched pattern ", pattern, "using ", path, "with ", f.Name())
			os.Remove(path)
		}
		return
	})
}

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
	defer cleanup(t, "/tmp/", "temp-subs*")
	fi, err := os.Lstat(subs.Source)
	if err != nil || fi.Size() == 0 {
		t.Fail()
	}
}

func TestAddingToSubscriptionsCreatesATimestampedFile(t *testing.T) {
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
	defer cleanup(t, "/tmp", "temp-subs*")
	files, _ := filepath.Glob("/tmp/temp-subs.*")
	if len(files) != 1 {
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
	defer cleanup(t, "/tmp", "temp-subs*")
	deserialized, _ := LoadSubscriptions(subs.Source)
	if len(subs.Info) != len(deserialized.Info) {
		t.Fail()
	}
}

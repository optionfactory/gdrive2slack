package gdrive2slack

import (
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/slack"
	"net/http"
	"os"
	"sync"
	"time"
)

func task(logger *Logger, client *http.Client, discardChannel chan string, waitGroup *sync.WaitGroup, configuration *google.OauthConfiguration, userState *UserState) {
	email := userState.GoogleUserInfo.Emails[0].Value
	slackUser := userState.SlackUserInfo.User
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[%s/%s] removing handler. reason: %v", email, slackUser, r)
			discardChannel <- email

		}
		waitGroup.Done()
	}()
	if userState.Gdrive.LargestChangeId == 0 {

		err := userState.Google.DoWithAccessToken(configuration, client, func(at string) (google.StatusCode, error) {
			return drive.LargestChangeId(client, userState.Gdrive, at)
		})
		if err != nil {
			logger.Warning("[%s/%s] %s", email, slackUser, err)
		}
		return
	}

	err := userState.Google.DoWithAccessToken(configuration, client, func(at string) (google.StatusCode, error) {
		return drive.DetectChanges(client, userState.Gdrive, at)
	})
	if err != nil {
		logger.Warning("[%s/%s] %s", email, slackUser, err)
		return
	}
	if len(userState.Gdrive.ChangeSet) > 0 {
		logger.Info("[%s/%s] @%v %v changes", email, slackUser, userState.Gdrive.LargestChangeId, len(userState.Gdrive.ChangeSet))
		message := CreateSlackMessage(userState)
		status, err := slack.PostMessage(client, message, userState.Slack.AccessToken)
		if status == slack.NotAuthed || status == slack.InvalidAuth || status == slack.AccountInactive {
			panic(err)
		}
		if status != slack.Ok {
			logger.Warning("[%s/%s] %s", email, slackUser, err)
		}

	}
}

func EventLoop(oauthConf *OauthConfiguration, logger *Logger, client *http.Client, registerChannel chan *UserState, discardChannel chan string, signalsChannel chan os.Signal) {
	var states, err = LoadUserStates("states.json")
	if err != nil {
		logger.Error("unreadable states", err)
		os.Exit(1)
	}
	var waitGroup sync.WaitGroup

	lastLoopTime := time.Time{}
	waitFor := time.Duration(0)
	for {
		if !lastLoopTime.IsZero() {
			waitFor = time.Duration(30)*time.Second - time.Now().Sub(lastLoopTime)
		}
		if waitFor < 0 {
			waitFor = time.Duration(1) * time.Second
		}
		select {
		case newState := <-registerChannel:
			states[newState.GoogleUserInfo.Emails[0].Value] = newState
		case email := <-discardChannel:
			delete(states, email)
		case s := <-signalsChannel:
			logger.Info("Exiting: got signal %v", s)
			SaveUserStates(states, "states.json")
			os.Exit(0)
		case <-time.After(waitFor):
			lastLoopTime = time.Now()
			for _, state := range states {
				waitGroup.Add(1)
				go task(logger, client, discardChannel, &waitGroup, oauthConf.Google, state)
			}
			waitGroup.Wait()
			logger.Info("Served %d clients", len(states))
		}
		SaveUserStates(states, "states.json")
	}
}

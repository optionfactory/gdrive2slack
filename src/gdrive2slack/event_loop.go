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

func task(logger *Logger, client *http.Client, discardChannel chan string, waitGroup *sync.WaitGroup, configuration *google.OauthConfiguration, subscription *Subscription, userState *UserState) {
	email := subscription.GoogleUserInfo.Email
	slackUser := subscription.SlackUserInfo.User
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[%s/%s] removing handler. reason: %v", email, slackUser, r)
			discardChannel <- email

		}
		waitGroup.Done()
	}()
	var err error
	if userState.Gdrive.LargestChangeId == 0 {

		userState.GoogleAccessToken, err = google.DoWithAccessToken(configuration, client, subscription.GoogleRefreshToken, userState.GoogleAccessToken, func(at string) (google.StatusCode, error) {
			return drive.LargestChangeId(client, userState.Gdrive, at)
		})
		if err != nil {
			logger.Warning("[%s/%s] %s", email, slackUser, err)
		}
		return
	}

	userState.GoogleAccessToken, err = google.DoWithAccessToken(configuration, client, subscription.GoogleRefreshToken, userState.GoogleAccessToken, func(at string) (google.StatusCode, error) {
		return drive.DetectChanges(client, userState.Gdrive, at)
	})
	if err != nil {
		logger.Warning("[%s/%s] %s", email, slackUser, err)
		return
	}
	if len(userState.Gdrive.ChangeSet) > 0 {
		logger.Info("[%s/%s] @%v %v changes", email, slackUser, userState.Gdrive.LargestChangeId, len(userState.Gdrive.ChangeSet))
		message := CreateSlackMessage(subscription, userState)
		status, err := slack.PostMessage(client, message, subscription.SlackAccessToken)
		if status == slack.NotAuthed || status == slack.InvalidAuth || status == slack.AccountInactive {
			panic(err)
		}
		if status != slack.Ok {
			logger.Warning("[%s/%s] %s", email, slackUser, err)
		}

	}
}

func EventLoop(oauthConf *OauthConfiguration, logger *Logger, client *http.Client, registerChannel chan *SubscriptionAndAccessToken, discardChannel chan string, signalsChannel chan os.Signal) {
	subscriptionsFileName := "subscriptions.json"
	userStates, subscriptions, err := LoadSubscriptions(subscriptionsFileName)
	if err != nil {
		logger.Error("unreadable subscriptions file", err)
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
		case subscriptionAndAccessToken := <-registerChannel:
			AddSubscription(userStates, subscriptions, subscriptionAndAccessToken.Subscription, subscriptionAndAccessToken.GoogleAccessToken)
			SaveSubscriptions(subscriptions, subscriptionsFileName)
		case email := <-discardChannel:
			RemoveSubscription(userStates, subscriptions, email)
			SaveSubscriptions(subscriptions, subscriptionsFileName)
		case s := <-signalsChannel:
			logger.Info("Exiting: got signal %v", s)
			os.Exit(0)
		case <-time.After(waitFor):
			lastLoopTime = time.Now()
			for k, subscription := range subscriptions {
				waitGroup.Add(1)
				go task(logger, client, discardChannel, &waitGroup, oauthConf.Google, subscription, userStates[k])
			}
			waitGroup.Wait()
			logger.Info("Served %d clients", len(subscriptions))
		}
	}
}

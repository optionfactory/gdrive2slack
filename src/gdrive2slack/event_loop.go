package gdrive2slack

import (
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/mailchimp"
	"github.com/optionfactory/gdrive2slack/slack"
	"net/http"
	"os"
	"sync"
	"time"
)

func task(logger *Logger, client *http.Client, discardChannel chan string, waitGroup *sync.WaitGroup, configuration *Configuration, subscription *Subscription, userState *UserState, version string) {
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

		userState.GoogleAccessToken, err = google.DoWithAccessToken(configuration.Google, client, subscription.GoogleRefreshToken, userState.GoogleAccessToken, func(at string) (google.StatusCode, error) {
			return drive.LargestChangeId(client, userState.Gdrive, at)
		})
		if err != nil {
			logger.Warning("[%s/%s] %s", email, slackUser, err)
		}
		return
	}

	userState.GoogleAccessToken, err = google.DoWithAccessToken(configuration.Google, client, subscription.GoogleRefreshToken, userState.GoogleAccessToken, func(at string) (google.StatusCode, error) {
		return drive.DetectChanges(client, userState.Gdrive, at)
	})
	if err != nil {
		logger.Warning("[%s/%s] %s", email, slackUser, err)
		return
	}
	if len(userState.Gdrive.ChangeSet) > 0 {
		logger.Info("[%s/%s] @%v %v changes", email, slackUser, userState.Gdrive.LargestChangeId, len(userState.Gdrive.ChangeSet))
		message := CreateSlackMessage(subscription, userState, version)
		status, err := slack.PostMessage(client, subscription.SlackAccessToken, message)
		if status == slack.NotAuthed || status == slack.InvalidAuth || status == slack.AccountInactive || status == slack.TokenRevoked {
			panic(err)
		}
		if status != slack.Ok {
			logger.Warning("[%s/%s] %s", email, slackUser, err)
		}
		if status == slack.ChannelNotFound {
			status, err = slack.PostMessage(client, subscription.SlackAccessToken, CreateSlackUnknownChannelMessage(subscription, configuration.Google.RedirectUri, message))
			if status == slack.NotAuthed || status == slack.InvalidAuth || status == slack.AccountInactive || status == slack.TokenRevoked {
				panic(err)
			}
			if status != slack.Ok {
				logger.Warning("[%s/%s] %s", email, slackUser, err)
			}
		}
	}
}

func EventLoop(configuration *Configuration, logger *Logger, client *http.Client, registerChannel chan *SubscriptionAndAccessToken, discardChannel chan string, signalsChannel chan os.Signal, version string) {
	subscriptionsFileName := "subscriptions.json"
	userStates, subscriptions, err := LoadSubscriptions(subscriptionsFileName)
	if err != nil {
		logger.Error("unreadable subscriptions file: %s", err)
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
			email := subscriptionAndAccessToken.Subscription.GoogleUserInfo.Email
			logger.Info("+subscription: %s", email)
			AddSubscription(userStates, subscriptions, subscriptionAndAccessToken.Subscription, subscriptionAndAccessToken.GoogleAccessToken)
			SaveSubscriptions(subscriptions, subscriptionsFileName)
			go func() {
				if configuration.Mailchimp.IsMailchimpConfigured() {
					error := mailchimp.Subscribe(configuration.Mailchimp, client, &mailchimp.SubscriptionRequest{
						Email:     email,
						FirstName: subscriptionAndAccessToken.Subscription.GoogleUserInfo.GivenName,
						LastName:  subscriptionAndAccessToken.Subscription.GoogleUserInfo.FamilyName,
					})
					if error != nil {
						logger.Warning("mailchimp/subscribe@%s %s", email, error)
					}
				}
			}()
		case email := <-discardChannel:
			logger.Info("-subscription: %s", email)
			RemoveSubscription(userStates, subscriptions, email)
			SaveSubscriptions(subscriptions, subscriptionsFileName)
			go func() {
				if configuration.Mailchimp.IsMailchimpConfigured() {
					error := mailchimp.Unsubscribe(configuration.Mailchimp, client, email)
					if error != nil {
						logger.Warning("mailchimp/unsubscribe@%s %s", email, error)
					}
				}
			}()
		case s := <-signalsChannel:
			logger.Info("Exiting: got signal %v", s)
			os.Exit(0)
		case <-time.After(waitFor):
			lastLoopTime = time.Now()
			for k, subscription := range subscriptions {
				waitGroup.Add(1)
				go task(logger, client, discardChannel, &waitGroup, configuration, subscription, userStates[k], version)
			}
			waitGroup.Wait()
			logger.Info("Served %d clients", len(subscriptions))
		}
	}
}

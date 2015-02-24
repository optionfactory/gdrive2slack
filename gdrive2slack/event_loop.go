package gdrive2slack

import (
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/mailchimp"
	"github.com/optionfactory/gdrive2slack/slack"
	"os"
	"sync"
	"time"
)

func EventLoop(env *Environment) {
	subscriptions, err := LoadSubscriptions("subscriptions.json")
	if err != nil {
		env.Logger.Error("unreadable subscriptions file: %s", err)
		os.Exit(1)
	}

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
		case subscriptionAndAccessToken := <-env.RegisterChannel:
			subscription := subscriptionAndAccessToken.Subscription
			subscriptions.Add(subscription, subscriptionAndAccessToken.GoogleAccessToken)
			if subscriptions.Contains(subscription.GoogleUserInfo.Email) {
				env.Logger.Info("*subscription: %s '%s' '%s'", subscription.GoogleUserInfo.Email, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName)
			} else {
				env.Logger.Info("+subscription: %s '%s' '%s'", subscription.GoogleUserInfo.Email, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName)
				go mailchimpRegistrationTask(env, subscription)
			}
		case email := <-env.DiscardChannel:
			subscription := subscriptions.Remove(email)
			env.Logger.Info("-subscription: %s '%s' '%s'", subscription.GoogleUserInfo.Email, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName)
			go mailchimpDeregistrationTask(env, subscription)
		case s := <-env.SignalsChannel:
			env.Logger.Info("Exiting: got signal %v", s)
			os.Exit(0)
		case <-time.After(waitFor):
			lastLoopTime = time.Now()
			var waitGroup sync.WaitGroup
			for k, subscription := range subscriptions.Info {
				waitGroup.Add(1)
				go serveUserTask(env, &waitGroup, subscription, subscriptions.States[k])
			}
			waitGroup.Wait()
			env.Logger.Info("Served %d clients", len(subscriptions.Info))
		}
	}
}

func serveUserTask(env *Environment, waitGroup *sync.WaitGroup, subscription *Subscription, userState *UserState) {
	email := subscription.GoogleUserInfo.Email
	slackUser := subscription.SlackUserInfo.User
	defer func() {
		if r := recover(); r != nil {
			env.Logger.Error("[%s/%s] removing handler. reason: %v", email, slackUser, r)
			env.DiscardChannel <- email

		}
		waitGroup.Done()
	}()
	var err error
	if userState.Gdrive.LargestChangeId == 0 {

		userState.GoogleAccessToken, err = google.DoWithAccessToken(env.Configuration.Google, env.HttpClient, subscription.GoogleRefreshToken, userState.GoogleAccessToken, func(at string) (google.StatusCode, error) {
			return drive.LargestChangeId(env.HttpClient, userState.Gdrive, at)
		})
		if err != nil {
			env.Logger.Warning("[%s/%s] %s", email, slackUser, err)
		}
		return
	}

	userState.GoogleAccessToken, err = google.DoWithAccessToken(env.Configuration.Google, env.HttpClient, subscription.GoogleRefreshToken, userState.GoogleAccessToken, func(at string) (google.StatusCode, error) {
		return drive.DetectChanges(env.HttpClient, userState.Gdrive, at)
	})
	if err != nil {
		env.Logger.Warning("[%s/%s] %s", email, slackUser, err)
		return
	}
	if len(userState.Gdrive.ChangeSet) > 0 {
		env.Logger.Info("[%s/%s] @%v %v changes", email, slackUser, userState.Gdrive.LargestChangeId, len(userState.Gdrive.ChangeSet))
		message := CreateSlackMessage(subscription, userState, env.Version)
		status, err := slack.PostMessage(env.HttpClient, subscription.SlackAccessToken, message)
		if status == slack.NotAuthed || status == slack.InvalidAuth || status == slack.AccountInactive || status == slack.TokenRevoked {
			panic(err)
		}
		if status != slack.Ok {
			env.Logger.Warning("[%s/%s] %s", email, slackUser, err)
		}
		if status == slack.ChannelNotFound {
			status, err = slack.PostMessage(env.HttpClient, subscription.SlackAccessToken, CreateSlackUnknownChannelMessage(subscription, env.Configuration.Google.RedirectUri, message))
			if status == slack.NotAuthed || status == slack.InvalidAuth || status == slack.AccountInactive || status == slack.TokenRevoked {
				panic(err)
			}
			if status != slack.Ok {
				env.Logger.Warning("[%s/%s] %s", email, slackUser, err)
			}
		}
	}
}

func mailchimpRegistrationTask(env *Environment, subscription *Subscription) {
	defer mailchimpRecover(env, subscription, "registration")
	if !env.Configuration.Mailchimp.IsMailchimpConfigured() {
		return
	}
	error := mailchimp.Subscribe(env.Configuration.Mailchimp, env.HttpClient, &mailchimp.SubscriptionRequest{
		Email:     subscription.GoogleUserInfo.Email,
		FirstName: subscription.GoogleUserInfo.GivenName,
		LastName:  subscription.GoogleUserInfo.FamilyName,
	})
	if error != nil {
		env.Logger.Warning("mailchimp/subscribe@%s %s", subscription.GoogleUserInfo.Email, error)
	}
}

func mailchimpDeregistrationTask(env *Environment, subscription *Subscription) {
	defer mailchimpRecover(env, subscription, "deregistration")
	if !env.Configuration.Mailchimp.IsMailchimpConfigured() {
		return
	}
	error := mailchimp.Unsubscribe(env.Configuration.Mailchimp, env.HttpClient, subscription.GoogleUserInfo.Email)
	if error != nil {
		env.Logger.Warning("mailchimp/unsubscribe@%s %s", subscription.GoogleUserInfo.Email, error)
	}
}

func mailchimpRecover(env *Environment, subscription *Subscription, task string) {
	if r := recover(); r != nil {
		env.Logger.Warning("[%s/%s] unexpected error in mailchimp %s task: %v", subscription.GoogleUserInfo.Email, subscription.SlackUserInfo.User, task, r)
	}
}

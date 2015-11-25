package gdrive2slack

import (
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/mailchimp"
	"github.com/optionfactory/gdrive2slack/slack"
	"os"
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
			alreadySubscribed := subscriptions.Contains(subscription.GoogleUserInfo.Email)
			subscriptions.Add(subscription, subscriptionAndAccessToken.GoogleAccessToken)
			if alreadySubscribed {
				env.Logger.Info("[%s/%s] *subscription: '%s' '%s'", subscription.GoogleUserInfo.Email, subscription.SlackUserInfo.User, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName)
			} else {
				env.Logger.Info("[%s/%s] +subscription: '%s' '%s'", subscription.GoogleUserInfo.Email, subscription.SlackUserInfo.User, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName)
				go mailchimpRegistrationTask(env, subscription)
			}
		case email := <-env.DiscardChannel:
			subscription, message, removed := subscriptions.HandleFailure(email)
			if removed {
				env.Logger.Info("[%s/%s] -subscription: '%s' '%s' %s", email, subscription.SlackUserInfo.User, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName, message)
				go mailchimpDeregistrationTask(env, subscription)
			} else {
				env.Logger.Info("[%s/%s] !subscription: '%s' '%s' %s", email, subscription.SlackUserInfo.User, subscription.GoogleUserInfo.GivenName, subscription.GoogleUserInfo.FamilyName, message)
			}
		case s := <-env.SignalsChannel:
			env.Logger.Info("Exiting: got signal %v", s)
			os.Exit(0)
		case <-time.After(waitFor):
			lastLoopTime = time.Now()

			subsLen := len(subscriptions.Info)
			requests := make(chan *SubscriptionAndUserState, subsLen)
			responses := make(chan bool, subsLen)

			for w := 0; w != env.Configuration.Workers; w++ {
				go Worker(w, env, requests, responses)
			}

			for k, subscription := range subscriptions.Info {
				requests <- &SubscriptionAndUserState{
					subscription,
					subscriptions.States[k],
				}
			}
			close(requests)

			for r := 0; r != subsLen; r++ {
				<-responses
			}
			env.Logger.Info("Served %d clients", subsLen)
		}
	}
}

type SubscriptionAndUserState struct {
	Subscription *Subscription
	UserState    *UserState
}

func Worker(id int, env *Environment, subAndStates <-chan *SubscriptionAndUserState, results chan<- bool) {
	for subAndState := range subAndStates {
		serveUserTask(env, subAndState.Subscription, subAndState.UserState)
		results <- true
	}
}

func serveUserTask(env *Environment, subscription *Subscription, userState *UserState) {
	email := subscription.GoogleUserInfo.Email
	slackUser := subscription.SlackUserInfo.User
	defer func() {
		if r := recover(); r != nil {
			env.Logger.Warning("[%s/%s] recovering. reason: %v", email, slackUser, r)
			env.DiscardChannel <- email
		}
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

	if len(userState.Gdrive.ChangeSet) == 0 {
		return
	}
	statusCode, err, folders := drive.FetchFolders(env.HttpClient, userState.GoogleAccessToken)
	if statusCode != google.Ok {
		env.Logger.Warning("[%s/%s] while fetching folders: %s", email, slackUser, err)
		return
	}
	message := CreateSlackMessage(subscription, userState, folders, env.Version)
	if len(message.Attachments) == 0 {
		return
	}

	env.Logger.Info("[%s/%s] @%v %v changes", email, slackUser, userState.Gdrive.LargestChangeId, len(message.Attachments))

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

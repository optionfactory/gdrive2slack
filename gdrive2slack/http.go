package gdrive2slack

import (
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	"net/http"
)

type Request struct {
	GoogleCode string   `json:"g"`
	SlackCode  string   `json:"s"`
	Channel    string   `json:"c"`
	FolderIds  []string `json:"fids"`
	FolderName string   `json:"fn"`
}

type ErrResponse struct {
	Error string `json:"error"`
}

func ServeHttp(env *Environment) {
	r := martini.NewRouter()
	mr := martini.New()
	mr.Use(martini.Recovery())
	mr.Use(martini.Static("public", martini.StaticOptions{
		SkipLogging: true,
	}))
	mr.MapTo(r, (*martini.Routes)(nil))
	mr.Action(r.Handle)
	m := &martini.ClassicMartini{mr, r}
	m.Use(render.Renderer())

	m.Get("/", func(renderer render.Render, req *http.Request) {
		renderer.HTML(200, "index", env)
	})
	m.Put("/", func(renderer render.Render, req *http.Request) {
		handleSubscriptionRequest(env, renderer, req)
	})
	m.RunOnAddr(env.Configuration.BindAddress)
}

func handleSubscriptionRequest(env *Environment, renderer render.Render, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var r Request
	err := decoder.Decode(&r)
	if err != nil {
		renderer.JSON(400, &ErrResponse{err.Error()})
		return
	}
	if r.GoogleCode == "" {
		renderer.JSON(400, &ErrResponse{"Invalid oauth code for google"})
		return
	}
	if r.SlackCode == "" {
		renderer.JSON(400, &ErrResponse{"Invalid oauth code for slack"})
		return
	}
	if r.Channel == "" {
		r.Channel = "#general"
	}
	googleRefreshToken, googleAccessToken, status, err := google.NewAccessToken(env.Configuration.Google, env.HttpClient, r.GoogleCode)
	if status != google.Ok {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}
	slackAccessToken, ostatus, err := slack.NewAccessToken(env.Configuration.Slack, env.HttpClient, r.SlackCode)
	if ostatus != slack.OauthOk {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}
	gUserInfo, status, err := userinfo.GetUserInfo(env.HttpClient, googleAccessToken)
	if status != google.Ok {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}
	sUserInfo, sstatus, err := slack.GetUserInfo(env.HttpClient, slackAccessToken)
	if sstatus != slack.Ok {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}

	welcomeMessage := CreateSlackWelcomeMessage(r.Channel, env.Configuration.Google.RedirectUri, sUserInfo, env.Version)
	cstatus, err := slack.PostMessage(env.HttpClient, slackAccessToken, welcomeMessage)

	env.RegisterChannel <- &SubscriptionAndAccessToken{
		Subscription: &Subscription{
			r.Channel,
			slackAccessToken,
			googleRefreshToken,
			gUserInfo,
			sUserInfo,
			r.FolderIds,
		},
		GoogleAccessToken: googleAccessToken,
	}

	renderer.JSON(200, map[string]interface{}{
		"user":         gUserInfo,
		"channelFound": cstatus == slack.Ok,
	})

}

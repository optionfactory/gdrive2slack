package gdrive2slack

import (
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/optionfactory/gdrive2slack/google"
	"github.com/optionfactory/gdrive2slack/google/userinfo"
	"github.com/optionfactory/gdrive2slack/slack"
	"net/http"
)

type Request struct {
	GoogleCode string `json:"g"`
	SlackCode  string `json:"s"`
	Channel    string `json:"c"`
}

type ErrResponse struct {
	Error string `json:"error"`
}

func ServeHttp(client *http.Client, registerChannel chan *SubscriptionAndAccessToken, configuration *Configuration, version string) {
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

	indexData := make(map[string]interface{})
	indexData["Configuration"] = configuration
	indexData["Version"] = version

	m.Get("/", func(renderer render.Render, req *http.Request) {
		renderer.HTML(200, "index", indexData)
	})
	m.Put("/", func(renderer render.Render, req *http.Request) {
		handleSubscriptionRequest(client, registerChannel, configuration, renderer, req)
	})
	m.RunOnAddr(configuration.BindAddress)
}

func handleSubscriptionRequest(client *http.Client, registerChannel chan *SubscriptionAndAccessToken, configuration *Configuration, renderer render.Render, req *http.Request) {
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
	googleRefreshToken, googleAccessToken, status, err := google.NewAccessToken(configuration.Google, client, r.GoogleCode)
	if status != google.Ok {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}
	slackAccessToken, ostatus, err := slack.NewAccessToken(configuration.Slack, client, r.SlackCode)
	if ostatus != slack.OauthOk {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}
	gUserInfo, status, err := userinfo.GetUserInfo(client, googleAccessToken)
	if status != google.Ok {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}
	sUserInfo, sstatus, err := slack.GetUserInfo(client, slackAccessToken)
	if sstatus != slack.Ok {
		renderer.JSON(500, &ErrResponse{err.Error()})
		return
	}

	cstatus, err := slack.PostMessage(client, slackAccessToken, &slack.Message{
		Channel:  r.Channel,
		Username: "Google Drive",
		Text:     fmt.Sprintf("A <%s|GDrive2Slack> integration has been configured by <@%s|%s>. Activities on Google Drive documents will be notified here.", configuration.Google.RedirectUri, sUserInfo.UserId, sUserInfo.User),
		IconUrl:  "http://gdrive2slack.optionfactory.net/gdrive2slack.png",
	})

	registerChannel <- &SubscriptionAndAccessToken{
		Subscription: &Subscription{
			r.Channel,
			slackAccessToken,
			googleRefreshToken,
			gUserInfo,
			sUserInfo,
		},
		GoogleAccessToken: googleAccessToken,
	}

	renderer.JSON(200, map[string]interface{}{
		"user":         gUserInfo,
		"channelFound": cstatus == slack.Ok,
	})

}

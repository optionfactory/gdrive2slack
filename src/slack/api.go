package slack

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type StatusCode int

const (
	Ok StatusCode = iota
	CannotConnect
	CannotDeserialize
	ChannelNotFound
	IsArchived
	MsgTooLong
	NoText
	RateLimited
	NotAuthed
	InvalidAuth
	TokenRevoked
	AccountInactive
	UserIsBot
	UnknownError
)

func (e StatusCode) String() string {
	return statusCodes[e]
}
func (e StatusCode) Error() string {
	return e.String()
}

func NewStatusCodeFromError(ec string) StatusCode {
	v, ok := errorLabelToStatusCode[ec]
	if ok {
		return v
	}
	return UnknownError
}

var errorLabelToStatusCode = map[string]StatusCode{
	"channel_not_found": ChannelNotFound,
	"is_archived":       IsArchived,
	"msg_too_long":      MsgTooLong,
	"no_text":           NoText,
	"rate_limited":      RateLimited,
	"not_authed":        NotAuthed,
	"invalid_auth":      InvalidAuth,
	"token_revoked":     TokenRevoked,
	"account_inactive":  AccountInactive,
	"user_is_bot":       UserIsBot,
}

var statusCodes = []string{
	Ok:                "ok",
	CannotConnect:     "cannot_connect",
	CannotDeserialize: "cannot_deserialize",
	ChannelNotFound:   "channel_not_found",
	IsArchived:        "is_archived",
	MsgTooLong:        "msg_too_long",
	NoText:            "no_text",
	RateLimited:       "rate_limited",
	NotAuthed:         "not_authed",
	InvalidAuth:       "invalid_auth",
	TokenRevoked:      "token_revoked",
	AccountInactive:   "account_inactive",
	UserIsBot:         "user_is_bot",
	UnknownError:      "unknown_error",
}

type Attachment struct {
	Fallback string  `json:"fallback"`
	Color    string  `json:"color"`
	Fields   []Field `json:"fields"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type Message struct {
	Channel     string       `json:"channel"`
	Username    string       `json:"username"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
	IconUrl     string       `json:"icon_url"`
}

type PostMessageResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

type UserInfo struct {
	Url    string `json:"url"`
	TeamId string `json:"team_id"`
	Team   string `json:"team"`
	UserId string `json:"user_id"`
	User   string `json:"user"`
}

type userInfoResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
	*UserInfo
}

func GetUserInfo(client *http.Client, accessToken string) (*UserInfo, StatusCode, error) {
	response, err := client.PostForm("https://slack.com/api/auth.test", url.Values{
		"token": {accessToken},
	})
	if err != nil {
		return nil, CannotConnect, err
	}
	defer response.Body.Close()
	var self = new(userInfoResponse)
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return nil, CannotDeserialize, err
	}
	if !self.Ok {
		return nil, NewStatusCodeFromError(self.Error), errors.New(self.Error)
	}
	return self.UserInfo, Ok, nil
}

func PostMessage(client *http.Client, message *Message, accessToken string) (StatusCode, error) {
	payload, _ := json.Marshal(message.Attachments)
	response, err := client.PostForm("https://slack.com/api/chat.postMessage", url.Values{
		"token":       {accessToken},
		"channel":     {message.Channel},
		"username":    {message.Username},
		"text":        {message.Text},
		"icon_url":    {message.IconUrl},
		"attachments": {string(payload)},
	})
	if err != nil {
		return CannotConnect, err
	}
	defer response.Body.Close()
	var self = new(PostMessageResponse)
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return CannotDeserialize, err
	}
	if !self.Ok {
		return NewStatusCodeFromError(self.Error), errors.New(self.Error)
	}
	return Ok, nil
}

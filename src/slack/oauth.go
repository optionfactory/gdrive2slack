package slack

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type OauthStatusCode int

const (
	OauthOk OauthStatusCode = iota
	OauthCannotConnect
	OauthCannotDeserialize
	OauthInvalidClientId
	OauthBadClientSecret
	OauthInvalidCode
	OauthBadRedirectUri
	OauthUnknownError
)

var errorLabelToOauthStatusCode = map[string]OauthStatusCode{
	"invalid_client_id": OauthInvalidClientId,
	"bad_client_secret": OauthBadClientSecret,
	"invalid_code":      OauthInvalidCode,
	"bad_redirect_uri":  OauthBadRedirectUri,
}

var oauthStatusCodes = []string{
	OauthOk:                "ok",
	OauthCannotConnect:     "cannot_connect",
	OauthCannotDeserialize: "cannot_deserialize",
	OauthInvalidClientId:   "invalid_client_id",
	OauthBadClientSecret:   "bad_client_secret",
	OauthInvalidCode:       "invalid_code",
	OauthBadRedirectUri:    "bad_redirect_uri",
	OauthUnknownError:      "unknown_error",
}

func (e OauthStatusCode) String() string {
	return oauthStatusCodes[e]
}
func (e OauthStatusCode) Error() string {
	return e.String()
}

func NewOauthStatusCodeFromError(ec string) OauthStatusCode {
	v, ok := errorLabelToOauthStatusCode[ec]
	if ok {
		return v
	}
	return OauthUnknownError
}

type OauthConfiguration struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthUri      string `json:"auth_uri"`
	TokenUri     string `json:"token_uri"`
	RedirectUri  string `json:"redirect_uri"`
}

type OauthTokenResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
	*OauthState
}

type OauthState struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
}

func NewAccessToken(conf *OauthConfiguration, client *http.Client, code string) (*OauthState, OauthStatusCode, error) {
	response, err := client.PostForm(conf.TokenUri, url.Values{
		"code":          {code},
		"client_id":     {conf.ClientId},
		"client_secret": {conf.ClientSecret},
		"redirect_uri":  {conf.RedirectUri},
	})
	if err != nil {
		return nil, OauthCannotConnect, err
	}
	defer response.Body.Close()
	var self = new(OauthTokenResponse)
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return nil, OauthCannotDeserialize, err
	}
	if !self.Ok {
		return nil, NewOauthStatusCodeFromError(self.Error), errors.New(self.Error)
	}
	return self.OauthState, OauthOk, nil
}

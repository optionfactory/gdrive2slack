package google

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

type ErrorResponse struct {
	Errors  []Error `json:"errors"`
	Code    uint    `json:"code"`
	Message string  `json:"message"`
}

func (self *ErrorResponse) Error() string {
	bytea, err := json.Marshal(self)
	if err != nil {
		panic(err)
	}
	return string(bytea)
}

type Error struct {
	Domain       string `json:"domain"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	LocationType string `json:"locationType"`
	Location     string `json:"location"`
}

type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	v, err := time.Parse("2006-01-02T15:04:05.000Z", string(b[1:len(b)-1]))
	if err != nil {
		return err
	}
	*t = Timestamp{v}
	return nil
}

func (t *Timestamp) Gte(others ...Timestamp) bool {
	for _, other := range others {
		if !t.Time.Equal(other.Time) && !t.Time.After(other.Time) {
			return false
		}
	}
	return true
}

type OauthConfiguration struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthUri      string `json:"auth_uri"`
	TokenUri     string `json:"token_uri"`
	RedirectUri  string `json:"redirect_uri"`
}

type OauthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type OauthState struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

func NewAccessToken(conf *OauthConfiguration, client *http.Client, code string) (*OauthState, StatusCode, error) {
	response, err := client.PostForm(conf.TokenUri, url.Values{
		"code":          {code},
		"client_id":     {conf.ClientId},
		"client_secret": {conf.ClientSecret},
		"redirect_uri":  {conf.RedirectUri},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, CannotConnect, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		oauthError := &OauthError{}
		err = json.NewDecoder(response.Body).Decode(oauthError)
		if err != nil {
			return nil, CannotDeserialize, err
		}
		if response.StatusCode >= 500 {
			return nil, ServerError, errors.New(oauthError.ErrorDescription)
		}
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return nil, Unauthorized, errors.New(oauthError.ErrorDescription)
		}
		return nil, ApiError, errors.New(oauthError.ErrorDescription)
	}
	var self = new(OauthState)
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return nil, CannotDeserialize, err
	}
	return self, Ok, nil
}

func (self *OauthState) RefreshAccessToken(conf *OauthConfiguration, client *http.Client) (StatusCode, error) {
	response, err := client.PostForm(conf.TokenUri, url.Values{
		"client_secret": {conf.ClientSecret},
		"client_id":     {conf.ClientId},
		"refresh_token": {self.RefreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return CannotConnect, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		oauthError := &OauthError{}
		err = json.NewDecoder(response.Body).Decode(oauthError)
		if err != nil {
			return CannotDeserialize, err
		}
		if response.StatusCode >= 500 {
			return ServerError, errors.New(oauthError.ErrorDescription)
		}
		if response.StatusCode == 400 && oauthError.Error == "invalid_grant" {
			return Unauthorized, errors.New(oauthError.Error)
		}
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return Unauthorized, errors.New(oauthError.ErrorDescription)
		}
		return ApiError, errors.New(oauthError.ErrorDescription)
	}
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return CannotDeserialize, err
	}
	return Ok, nil
}

type callback func(string) (StatusCode, error)

func (self *OauthState) DoWithAccessToken(conf *OauthConfiguration, client *http.Client, cb callback) error {
	code, err := cb(self.AccessToken)
	if code == Ok {
		return nil
	}
	if code == Unauthorized {
		code, err = self.RefreshAccessToken(conf, client)
		if code == Unauthorized {
			panic(err)
		}
		if code != Ok {
			return err
		}
	}
	code, err = cb(self.AccessToken)
	if code == Unauthorized {
		panic(err)
	}
	return err
}

type StatusCode int

const (
	Ok StatusCode = iota
	CannotConnect
	CannotDeserialize
	Unauthorized
	ServerError
	ApiError
)

var errorNames = []string{
	Ok:                "Ok",
	CannotConnect:     "Connection problem",
	CannotDeserialize: "Cannot deserialize response",
	Unauthorized:      "Unauthorized",
	ServerError:       "Server error",
	ApiError:          "Api error",
}

func (e StatusCode) String() string {
	return errorNames[e]
}

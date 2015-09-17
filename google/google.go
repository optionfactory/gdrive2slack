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
	ApiKey       string `json:"api_key"`
	RedirectUri  string `json:"redirect_uri"`
}

type OauthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type OauthState struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewAccessToken(conf *OauthConfiguration, client *http.Client, code string) (string, string, StatusCode, error) {
	response, err := client.PostForm("https://accounts.google.com/o/oauth2/token", url.Values{
		"code":          {code},
		"client_id":     {conf.ClientId},
		"client_secret": {conf.ClientSecret},
		"redirect_uri":  {conf.RedirectUri},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return "", "", CannotConnect, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		oauthError := &OauthError{}
		err = json.NewDecoder(response.Body).Decode(oauthError)
		if err != nil {
			return "", "", CannotDeserialize, err
		}
		if response.StatusCode >= 500 {
			return "", "", ServerError, errors.New(oauthError.ErrorDescription)
		}
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return "", "", Unauthorized, errors.New(oauthError.ErrorDescription)
		}
		return "", "", ApiError, errors.New(oauthError.ErrorDescription)
	}
	var self = new(OauthState)
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return "", "", CannotDeserialize, err
	}
	return self.RefreshToken, self.AccessToken, Ok, nil
}

func RefreshAccessToken(conf *OauthConfiguration, client *http.Client, refreshToken string) (string, StatusCode, error) {
	response, err := client.PostForm("https://accounts.google.com/o/oauth2/token", url.Values{
		"client_secret": {conf.ClientSecret},
		"client_id":     {conf.ClientId},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return "", CannotConnect, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		oauthError := &OauthError{}
		err = json.NewDecoder(response.Body).Decode(oauthError)
		if err != nil {
			return "", CannotDeserialize, err
		}
		if response.StatusCode >= 500 {
			return "", ServerError, errors.New(oauthError.ErrorDescription)
		}
		if response.StatusCode == 400 && oauthError.Error == "invalid_grant" {
			return "", Unauthorized, errors.New(oauthError.Error)
		}
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return "", Unauthorized, errors.New(oauthError.ErrorDescription)
		}
		return "", ApiError, errors.New(oauthError.ErrorDescription)
	}
	var self = new(OauthState)
	err = json.NewDecoder(response.Body).Decode(self)
	if err != nil {
		return "", CannotDeserialize, err
	}
	return self.AccessToken, Ok, nil
}

type callback func(string) (StatusCode, error)

func DoWithAccessToken(conf *OauthConfiguration, client *http.Client, refreshToken string, accessToken string, cb callback) (string, error) {
	code, err := cb(accessToken)
	if code == Ok {
		return accessToken, nil
	}
	if code == Unauthorized {
		accessToken, code, err = RefreshAccessToken(conf, client, refreshToken)
		if code == Unauthorized {
			panic(err)
		}
		if code != Ok {
			return accessToken, err
		}
	}
	code, err = cb(accessToken)
	if code == Unauthorized {
		panic(err)
	}
	return accessToken, err
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

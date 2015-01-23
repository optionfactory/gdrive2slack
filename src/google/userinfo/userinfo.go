package userinfo

import (
	"github.com/optionfactory/gdrive2slack/google"	
	"net/url"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"errors"
)

type response struct {
	Error *google.ErrorResponse `json:"error"`
	*UserInfo
}

type UserInfo struct {
	DisplayName string `json:"displayName"`
	Name Name `json:"name"`
	Emails []Email `json:"emails"`
}

type Name struct {
	GivenName string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type Email struct { 
	Value string `json:"value"`
}

func GetUserInfo(client *http.Client, accessToken string) (*UserInfo, google.StatusCode, error){
	u, _ := url.Parse("https://www.googleapis.com/plus/v1/people/me")
	q := u.Query()
	q.Set("fields", "name,displayName,emails")
	q.Set("userId", "me")
	u.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Add("Authorization", "Bearer " + accessToken)
	res, err := client.Do(req)
	if err != nil {
		return nil, google.CannotConnect, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	var deser = new(response)
	err = json.Unmarshal(body, &deser)
	if err != nil {
		return nil, google.CannotDeserialize, err
	}
	if deser.Error != nil {
		if deser.Error.Code == 401 {
			return nil, google.Unauthorized, errors.New(deser.Error.Message)
		}
		return nil, google.ApiError, errors.New(deser.Error.Message)
	}	
	return deser.UserInfo, google.Ok, nil
}
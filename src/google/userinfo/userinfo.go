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
	DisplayName string `json:"displayName"`
	Name name `json:"name"`
	Emails []email `json:"emails"`
}

type name struct {
	GivenName string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type email struct { 
	Value string `json:"value"`
}

type UserInfo struct {
	DisplayName string `json:"displayName"`
	GivenName string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Email string `json:"email"`
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
	userInfo := &UserInfo{
		DisplayName: deser.DisplayName, 
		GivenName: deser.Name.GivenName,
		FamilyName: deser.Name.FamilyName,
		Email: deser.Emails[0].Value,
	}
	return userInfo, google.Ok, nil
}
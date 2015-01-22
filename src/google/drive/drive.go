package drive

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"github.com/optionfactory/gdrive2slack/google"
	"errors"
)


type changes struct {
	LargestChangeId string         `json:"largestChangeId"`
	Items           []ChangeItem   `json:"items"`
	Error           *google.ErrorResponse `json:"error"`
}


type ChangeItem struct {
	Deleted    bool        `json:"deleted"`
	LastAction Action      `json:"-"`
	File       ChangedFile `json:"file"`
}


type ChangedFile struct {
	ExplicitlyTrashed bool      `json:"explicitlyTrashed"`
	LastModifyingUser User      `json:"lastModifyingUser"`
	AlternateLink     string    `json:"alternateLink"`
	MimeType          string    `json:"mimeType"`
	CreatedDate       google.Timestamp `json:"createdDate"`
	ModifiedDate      google.Timestamp `json:"modifiedDate"`
	SharedWithMeDate  google.Timestamp `json:"sharedWithMeDate"`
	Title             string    `json:"title"`
}


type Action int

const (
	Deleted Action = iota
	Created
	Modified
	Shared
	Viewed
)

var actionNames = []string{
	Deleted:  "Deleted",
	Created:  "Created",
	Modified: "Modified",
	Shared:   "Shared",
	Viewed:   "Viewed",
}

func (t Action) String() string {
	return actionNames[t]
}

func (t *ChangeItem) updateLastAction(timeRef time.Time) {
	var f = t.File
	var threshold = google.Timestamp{timeRef.Add(-time.Duration(10) * time.Minute)}
	if t.Deleted || f.ExplicitlyTrashed {
		t.LastAction = Deleted
		return
	}
	if f.CreatedDate.Gte(f.ModifiedDate, f.SharedWithMeDate, threshold) {
		t.LastAction = Created
		return
	}
	if f.ModifiedDate.Gte(f.SharedWithMeDate, threshold) {
		t.LastAction = Modified
		return
	}
	if !f.SharedWithMeDate.IsZero() && f.SharedWithMeDate.Gte(threshold) {
		t.LastAction = Shared
		return
	}
	t.LastAction = Viewed
}

type User struct {
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName`
}

type GracePeriodKey struct {
	FileTitle string
	LastModifyingUserEmail string
}


type State struct {
	LargestChangeId uint64
	InGracePeriod       map[GracePeriodKey]time.Time
	ChangeSet       []ChangeItem
}



func NewState() *State {
	return &State{
		InGracePeriod: make(map[GracePeriodKey]time.Time),
	}
}



func query(client *http.Client, state *State, accessToken string) (google.StatusCode, error) {
	var timeRef = time.Now();
	u, _ := url.Parse("https://www.googleapis.com/drive/v2/changes")
	q := u.Query()
	if state.LargestChangeId == 0 {
		q.Set("fields", "largestChangeId")
	} else {
		q.Set("fields", "largestChangeId,items(deleted,file(explicitlyTrashed,alternateLink,mimeType,createdDate,modifiedDate,sharedWithMeDate,title,lastModifyingUser(displayName,emailAddress)))")
		q.Set("startChangeId", strconv.FormatUint(state.LargestChangeId+1, 10))
	}
	q.Set("includeDeleted", "true")
	q.Set("includeSubscribed", "false")
	q.Set("maxResults", "100")
	u.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Add("Authorization", "Bearer "+accessToken)
	response, err := client.Do(req)
	if err != nil {
		return google.CannotConnect, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	var changes = new(changes)
	err = json.Unmarshal(body, &changes)

	if err != nil {
		return google.CannotDeserialize, err
	}
	if changes.Error != nil {
		if changes.Error.Code == 401 {
			return google.Unauthorized, errors.New(changes.Error.Message)
		}
		return google.ApiError, errors.New(changes.Error.Message)
	}
	state.LargestChangeId, err = strconv.ParseUint(changes.LargestChangeId, 10, 64)
	state.ChangeSet = make([]ChangeItem, 0, len(changes.Items))

	if len(changes.Items) == 0 {
		return google.Ok, nil
	}
	var threshold = timeRef.Add(time.Duration(-60) * time.Minute)
	for _, item := range changes.Items {
		item.updateLastAction(timeRef)
		if item.LastAction == Viewed || item.File.Title == ""{
			continue
		}
		k := GracePeriodKey{item.File.Title, item.File.LastModifyingUser.EmailAddress}
		notifiedAt, alreadyNotified := state.InGracePeriod[k]
		if !(alreadyNotified && notifiedAt.After(threshold) && item.LastAction != Deleted) {
			state.InGracePeriod[k] = timeRef
			state.ChangeSet = append(state.ChangeSet, item)
		}
	}
	for k, at := range state.InGracePeriod {
		if at.Before(threshold) {
			delete(state.InGracePeriod, k)
		}
	}
	return google.Ok, nil
}

func LargestChangeId(client *http.Client, state *State, accessToken string) (google.StatusCode, error) {
	return query(client, state, accessToken)
}

func DetectChanges(client *http.Client, state *State, accessToken string) (google.StatusCode, error) {
	return query(client, state, accessToken)
}

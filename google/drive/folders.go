package drive

import (
	"encoding/json"
	"errors"
	"github.com/optionfactory/gdrive2slack/google"
	"io/ioutil"
	"net/http"
	"net/url"
)

type folders struct {
	NextPageToken string                `json:"nextPageToken"`
	Items         []*folder             `json:"items"`
	Error         *google.ErrorResponse `json:"error"`
}

type folder struct {
	Id      string   `json:"id"`
	Title   string   `json:"title"`
	Parents []parent `json:"parents"`
}

type parent struct {
	Id string `json:"id"`
}

type Folder struct {
	Id   string
	Name string
	Path string
}

func fetchFolders(client *http.Client, accessToken string) (google.StatusCode, error, *folders) {
	u, _ := url.Parse("https://www.googleapis.com/drive/v2/files")
	q := u.Query()
	q.Set("corpus", "DOMAIN")
	q.Set("q", "mimeType = 'application/vnd.google-apps.folder'")
	q.Set("fields", "items(id,parents(id),title),nextPageToken")
	q.Set("maxResults", "1000")
	u.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Add("Authorization", "Bearer "+accessToken)
	response, err := client.Do(req)
	if err != nil {
		return google.CannotConnect, err, nil
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	var folders = new(folders)
	err = json.Unmarshal(body, &folders)

	if err != nil {
		return google.CannotDeserialize, err, nil
	}
	if folders.Error != nil {
		if folders.Error.Code == 401 {
			return google.Unauthorized, errors.New(folders.Error.Message), nil
		}
		return google.ApiError, errors.New(folders.Error.Message), nil
	}
	return google.Ok, nil, folders
}

func index(folders *folders) map[string]*folder {
	indexed := make(map[string]*folder)
	for _, f := range folders.Items {
		indexed[f.Id] = f
	}
	return indexed
}

func path(indexed map[string]*folder, id string) string {
	path := ""
	for {
		current, ok := indexed[id]
		if !ok {
			return path
		}
		path = current.Title + "/" + path
		id = current.Parents[0].Id
	}
	return path
}

func flatten(indexed map[string]*folder) []Folder {
	list := make([]Folder, 0)
	for _, f := range indexed {
		list = append(list, Folder{
			Id:   f.Id,
			Name: f.Title,
			Path: path(indexed, f.Parents[0].Id),
		})
	}
	return list
}

func ListFolders(client *http.Client, accessToken string) (google.StatusCode, error, []Folder) {
	status, err, fs := fetchFolders(client, accessToken)
	if status != google.Ok {
		return status, err, nil
	}
	return google.Ok, nil, flatten(index(fs))
}

func PathFor(client *http.Client, accessToken string, id string) (google.StatusCode, error, string) {
	status, err, fs := fetchFolders(client, accessToken)
	if status != google.Ok {
		return status, err, ""
	}
	return google.Ok, nil, path(index(fs), id)
}

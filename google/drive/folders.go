package drive

import (
	"encoding/json"
	"errors"
	"github.com/optionfactory/gdrive2slack/google"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Folders struct {
	inner map[string]*Folder
}

type folders struct {
	NextPageToken string                `json:"nextPageToken"`
	Items         []*folder             `json:"items"`
	Error         *google.ErrorResponse `json:"error"`
}

type folder struct {
	Id      string   `json:"id"`
	Title   string   `json:"title"`
	Parents []Parent `json:"parents"`
}

type Parent struct {
	Id string `json:"id"`
}

type Folder struct {
	Id        string
	Name      string
	Path      string
	ParentIds []string
}

// google prevents creating loops in folders, so we don't need to check for it.
func index(folders []*folder) *Folders {
	indexed := make(map[string]*Folder)
	for _, f := range folders {
		current := &Folder{
			Id:        f.Id,
			Name:      f.Title,
			Path:      "",
			ParentIds: make([]string, 0),
		}
		for _, parent := range f.Parents {
			current.ParentIds = append(current.ParentIds, parent.Id)
		}
		indexed[f.Id] = current
	}
	for id, folder := range indexed {
		path := ""
		id = folder.ParentIds[0]
		for {
			current, ok := indexed[id]
			if !ok {
				folder.Path = path
				break
			}
			if path != "" {
				path = current.Name + "/" + path
			} else {
				path = current.Name
			}
			id = current.ParentIds[0]
		}
	}
	return &Folders{
		inner: indexed,
	}
}

func FetchFolders(client *http.Client, accessToken string) (google.StatusCode, error, *Folders) {
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
	return google.Ok, nil, index(folders.Items)
}

func (self *Folders) List() []*Folder {
	list := make([]*Folder, 0)
	for _, f := range self.inner {
		list = append(list, f)
	}
	return list
}

func (self *Folders) PathFor(folderId string) (string, bool) {
	folder, contained := self.inner[folderId]
	if !contained {
		return "", contained
	}
	return folder.Path, contained
}

func (self *Folders) folderIsOrIsContainedIn(needle string, haystack string) bool {
	current, found := self.inner[needle]
	if !found {
		return false
	}
	if current.Id == haystack {
		return true
	}
	for _, needle = range current.ParentIds {
		if self.folderIsOrIsContainedIn(needle, haystack) {
			return true
		}
	}
	return false
}

func (self *Folders) FolderIsOrIsContainedInAny(folders []Parent, parentIds []string) bool {
	for _, folder := range folders {
		for _, parentId := range parentIds {
			if self.folderIsOrIsContainedIn(folder.Id, parentId) {
				return true
			}
		}
	}
	return false
}

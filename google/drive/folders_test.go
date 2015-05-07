package drive

import (
	_ "log"
	"testing"
)

func TestListOnEmptyFoldersYieldAnEmptySlice(t *testing.T) {
	f := &Folders{
		inner: make(map[string]*Folder),
	}

	got := f.List()
	if len(got) != 0 {
		t.Fail()
	}
}

func TestIndexingMergesPaths(t *testing.T) {

	f := []*folder{
		{
			Id:    "1",
			Title: "parent",
			Parents: []parent{
				{"0"},
			},
		},
		{
			Id:    "2",
			Title: "child",
			Parents: []parent{
				{"1"},
			},
		},
	}
	for _, item := range index(f).inner {
		t.Log(item)
		if item.Name == "child" && item.Path == "parent" {
			return
		}
	}
	t.Fail()
}

func TestListYieldsSameNumberOfInputFolders(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "something",
			Parents: []parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "somethingelse",
			Parents: []parent{
				{Id: "0"},
			},
		},
	}
	f := index(in)

	got := f.List()

	if len(got) != len(in) {
		t.Fail()
	}
}

func TestPathForRootYieldEmptyString(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "something",
			Parents: []parent{
				{Id: "0"},
			},
		},
	}
	f := index(in)
	if got, _ := f.PathFor("1"); got != "" {
		t.Fail()
	}
}

func TestFolderIsContainedInItself(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "something",
			Parents: []parent{
				{Id: "0"},
			},
		},
	}
	f := index(in)
	if !f.FolderIsOrIsContainedInAny([]string{"1"}, []string{"1"}) {
		t.Fail()
	}
}

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
			Parents: []Parent{
				{"0"},
			},
		},
		{
			Id:    "2",
			Title: "child",
			Parents: []Parent{
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
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "somethingelse",
			Parents: []Parent{
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
			Parents: []Parent{
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
			Parents: []Parent{
				{Id: "0"},
			},
		},
	}
	f := index(in)
	if !f.FolderIsOrIsContainedInAny([]Parent{{"1"}}, []string{"1"}) {
		t.Fail()
	}
}

func TestFolderIsContainedInParent(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "parent",
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "child",
			Parents: []Parent{
				{Id: "1"},
			},
		},
	}
	f := index(in)
	if !f.FolderIsOrIsContainedInAny([]Parent{{"2"}}, []string{"1"}) {
		t.Fail()
	}
}

func TestFolderIsContainedInAnyParent(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "parent",
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "parent",
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "3",
			Title: "child",
			Parents: []Parent{
				{Id: "1"}, {Id: "2"},
			},
		},
	}
	f := index(in)
	if !f.FolderIsOrIsContainedInAny([]Parent{{"3"}}, []string{"2"}) {
		t.Fail()
	}
}

func TestFolderIsContainedInAncestor(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "parent",
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "parent",
			Parents: []Parent{
				{Id: "1"},
			},
		},
		{
			Id:    "3",
			Title: "child",
			Parents: []Parent{
				{Id: "2"},
			},
		},
	}
	f := index(in)
	if !f.FolderIsOrIsContainedInAny([]Parent{{"3"}}, []string{"1"}) {
		t.Fail()
	}
}

func TestFolderIsContainedInAnySiblingAncestor(t *testing.T) {
	in := []*folder{
		{
			Id:    "1",
			Title: "parent",
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "1b",
			Title: "parent",
			Parents: []Parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "parent",
			Parents: []Parent{
				{Id: "1"}, {Id: "1b"},
			},
		},
		{
			Id:    "3",
			Title: "child",
			Parents: []Parent{
				{Id: "2"},
			},
		},
	}
	f := index(in)
	if !f.FolderIsOrIsContainedInAny([]Parent{{"3"}}, []string{"1b"}) {
		t.Fail()
	}
}

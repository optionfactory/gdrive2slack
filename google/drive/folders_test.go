package drive

import (
	_ "log"
	"testing"
)

func TestFlatteningAnEmptySliceYieldEmpty(t *testing.T) {
	got := flatten(index(&folders{
		Items: []*folder{},
	}))
	if len(got) != 0 {
		t.Fail()
	}
}

func TestFlattenMergePath(t *testing.T) {
	items := []*folder{
		{
			Id:    "1",
			Title: "root",
			Parents: []parent{
				{Id: "0"},
			},
		},
		{
			Id:    "2",
			Title: "child",
			Parents: []parent{
				{Id: "1"},
			},
		},
	}

	got := flatten(index(&folders{
		Items: items,
	}))
	for _, item := range got {
		if item.Name == "child" && item.Path == "root/" {
			return
		}
	}
	t.Fail()
}

func TestFlattenYieldsSameNumberOfFolders(t *testing.T) {
	items := []*folder{
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

	got := flatten(index(&folders{
		Items: items,
	}))
	if len(items) != len(got) {
		t.Fail()
	}
}

func TestPathForRootYieldEmptyString(t *testing.T) {
	items := []*folder{
		{
			Id:    "1",
			Title: "something",
			Parents: []parent{
				{Id: "0"},
			},
		},
	}
	got := path(index(&folders{
		Items: items,
	}), "0")
	if got != "" {
		t.Fail()
	}
}

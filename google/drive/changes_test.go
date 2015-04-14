package drive

import (
	_ "log"
	"testing"
)

func TestRegularOfficeFilesAreNotTemporary(t *testing.T) {
	if isTemporaryFile("test.xlsx") == true {
		t.Fail()
	}
}

func TestCanDetectOfficeTemporaryFiles(t *testing.T) {
	if isTemporaryFile("~$test.xlsx") == false {
		t.Fail()
	}
}

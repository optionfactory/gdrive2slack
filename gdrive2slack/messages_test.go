package gdrive2slack

import (
	"testing"
)

func TestInfixWithEmptyStringYieldsEmptyString(t *testing.T) {
	if "" != infixZeroWidthSpace("") {
		t.Fail()
	}
}

func TestInfixWithOneByteStringYieldsSource(t *testing.T) {
	if "A" != infixZeroWidthSpace("A") {
		t.Fail()
	}
}

func TestInfixOnOneRuneStringYieldsSource(t *testing.T) {
	if "昭" != infixZeroWidthSpace("昭") {
		t.Fail()
	}
}

func TestInfixOnMultiRuneStringYieldsSource(t *testing.T) {
	if "花\u200B子" != infixZeroWidthSpace("花子") {
		t.Fail()
	}
}

func TestPreventNotificationOnEmptyStringYieldsEmptyString(t *testing.T) {
	if "" != preventNotification("") {
		t.Fail()
	}
}

func TestPreventNotificationOnSingleWordCreateAnInfixWord(t *testing.T) {
	if infixZeroWidthSpace("花子") != preventNotification("花子") {
		t.Fail()
	}
}

func TestPreventNotificationOnMultipleWordsInfixEveryWord(t *testing.T) {
	if infixZeroWidthSpace("花子")+" "+infixZeroWidthSpace("عادلة") != preventNotification("花子 عادلة") {
		t.Fail()
	}
}

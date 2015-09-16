package gdrive2slack

import (
	"fmt"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/slack"
	"strings"
	"unicode/utf8"
)

var actionColors = []string{
	drive.Deleted:  "#ffcccc",
	drive.Created:  "#ccffcc",
	drive.Modified: "#ccccff",
	drive.Shared:   "#ccccff",
	drive.Viewed:   "#ccccff",
}

func infixZeroWidthSpace(source string) string {
	if utf8.RuneCountInString(source) < 2 {
		return source
	}
	firstRune, width := utf8.DecodeRuneInString(source)
	rest := source[width:]
	return string(firstRune) + "\u200B" + rest
}

func preventNotification(source string) string {
	split := strings.Split(source, " ")
	for i, word := range split {
		split[i] = infixZeroWidthSpace(word)
	}
	return strings.Join(split, " ")
}

func CreateSlackAttachment(change *drive.ChangeItem) *slack.Attachment {
	var editor string
	if len(change.File.LastModifyingUser.EmailAddress) > 0 && len(change.File.LastModifyingUser.DisplayName) > 0 {
		editor = fmt.Sprintf("<mailto:%s|%s>", change.File.LastModifyingUser.EmailAddress, preventNotification(change.File.LastModifyingUser.DisplayName))
	} else if len(change.File.LastModifyingUser.DisplayName) > 0 {
		editor = preventNotification(change.File.LastModifyingUser.DisplayName)
	} else {
		editor = "Unknown"
	}
	return &slack.Attachment{
		Fallback: fmt.Sprintf("Changes Detected to %s <%s|%s>", change.Type, change.File.AlternateLink, change.File.Title),
		Color:    actionColors[change.LastAction],
		Fields: []slack.Field{
			{
				Title: fmt.Sprintf("%s %s", change.LastAction, change.Type),
				Value: fmt.Sprintf("<%s|%s>", change.File.AlternateLink, change.File.Title),
				Short: true,
			},
			{
				Title: "Editor",
				Value: editor,
				Short: true,
			},
		},
	}
}

func CreateSlackMessage(subscription *Subscription, userState *UserState, folders *drive.Folders, version string) *slack.Message {
	var attachments = make([]slack.Attachment, 0, len(userState.Gdrive.ChangeSet))
	var roots = subscription.GoogleInterestingFolderIds
	for i := 0; i != len(userState.Gdrive.ChangeSet); i++ {
		if len(roots) == 0 || folders.FolderIsOrIsContainedInAny(userState.Gdrive.ChangeSet[i].File.Parents, roots) {
			attachments = append(attachments, *CreateSlackAttachment(&userState.Gdrive.ChangeSet[i]))
		}

	}
	return &slack.Message{
		Channel:     subscription.Channel,
		Username:    "Google Drive",
		Text:        fmt.Sprintf("Activity on gdrive (configured by @%s)", preventNotification(subscription.SlackUserInfo.User)),
		IconUrl:     fmt.Sprintf("http://gdrive2slack.optionfactory.net/gdrive2slack.png?ck=%s", version),
		Attachments: attachments,
	}
}

func CreateSlackWelcomeMessage(channel string, redirectUri string, sUserInfo *slack.UserInfo, version string) *slack.Message {
	return &slack.Message{
		Channel:  channel,
		Username: "Google Drive",
		Text:     fmt.Sprintf("A <%s|GDrive2Slack> integration has been configured by <@%s|%s>. Activities on Google Drive documents will be notified here.", redirectUri, sUserInfo.UserId, sUserInfo.User),
		IconUrl:  fmt.Sprintf("http://gdrive2slack.optionfactory.net/gdrive2slack.png?ck=%s", version),
	}
}

func CreateSlackUnknownChannelMessage(subscription *Subscription, redirectUri string, source *slack.Message) *slack.Message {
	nonExistentChannel := source.Channel
	return &slack.Message{
		Channel:     "@" + subscription.SlackUserInfo.User,
		Username:    "Google Drive",
		Text:        fmt.Sprintf("Hey <@%s|%s>, something is wrong: we can't find the slack channel %s: you should either create or <%s|change it>. Here is what happened in the meantime:", subscription.SlackUserInfo.User, subscription.SlackUserInfo.UserId, nonExistentChannel, redirectUri),
		IconUrl:     source.IconUrl,
		Attachments: source.Attachments,
	}
}

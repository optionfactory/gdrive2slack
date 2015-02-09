package gdrive2slack

import (
	"fmt"
	"github.com/optionfactory/gdrive2slack/google/drive"
	"github.com/optionfactory/gdrive2slack/slack"
)

var actionColors = []string{
	drive.Deleted:  "#ffcccc",
	drive.Created:  "#ccffcc",
	drive.Modified: "#ccccff",
	drive.Shared:   "#ccccff",
	drive.Viewed:   "#ccccff",
}

func CreateSlackAttachment(change *drive.ChangeItem) *slack.Attachment {
	var editor string
	if len(change.File.LastModifyingUser.EmailAddress) > 0 && len(change.File.LastModifyingUser.DisplayName) > 0 {
		editor = fmt.Sprintf("<mailto:%s|%s>", change.File.LastModifyingUser.EmailAddress, change.File.LastModifyingUser.DisplayName)
	} else if len(change.File.LastModifyingUser.DisplayName) > 0 {
		editor = change.File.LastModifyingUser.DisplayName
	} else {
		editor = "Unknown"
	}
	return &slack.Attachment{
		Fallback: fmt.Sprintf("Changes Detected to file <%s|%s>", change.File.AlternateLink, change.File.Title),
		Color:    actionColors[change.LastAction],
		Fields: []slack.Field{
			{
				Title: fmt.Sprintf("%s file", change.LastAction.String()),
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

func CreateSlackMessage(subscription *Subscription, userState *UserState) *slack.Message {

	var attachments = make([]slack.Attachment, 0, len(userState.Gdrive.ChangeSet))

	for i := 0; i != len(userState.Gdrive.ChangeSet); i++ {
		attachments = append(attachments, *CreateSlackAttachment(&userState.Gdrive.ChangeSet[i]))
	}

	return &slack.Message{
		Channel:     subscription.Channel,
		Username:    "Google Drive",
		Text:        fmt.Sprintf("Activity on gdrive (configured by <@%s|%s>)", subscription.SlackUserInfo.UserId, subscription.SlackUserInfo.User),
		IconUrl:     "http://gdrive2slack.optionfactory.net/gdrive2slack.png",
		Attachments: attachments,
	}
}

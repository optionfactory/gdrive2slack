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
	return &slack.Attachment{
		Fallback: fmt.Sprintf("Changes Detected to file: <%s|%s>", change.File.AlternateLink, change.File.Title),
		Color:    actionColors[change.LastAction],
		Fields: []slack.Field{
			{
				Title: fmt.Sprintf("%s file:", change.LastAction.String()),
				Value: fmt.Sprintf("<%s|%s>", change.File.AlternateLink, change.File.Title),
				Short: true,
			},
			{
				Title: "Editor",
				Value: fmt.Sprintf("<mailto:%s|%s>", change.File.LastModifyingUser.EmailAddress, change.File.LastModifyingUser.DisplayName),
				Short: true,
			},
		},
	}
}

func CreateSlackMessage(userState *UserState) *slack.Message {

	var attachments = make([]slack.Attachment, 0, len(userState.Gdrive.ChangeSet))

	for i := 0; i != len(userState.Gdrive.ChangeSet); i++ {
		attachments = append(attachments, *CreateSlackAttachment(&userState.Gdrive.ChangeSet[i]))
	}

	return &slack.Message{
		Channel:     userState.Channel,
		Username:    "Google Drive",
		Text:        fmt.Sprintf("Activity on google drive: (hook for <mailto:%s|%s> → <@%s|%s>)", userState.GoogleUserInfo.Emails[0].Value, userState.GoogleUserInfo.DisplayName, userState.SlackUserInfo.UserId, userState.SlackUserInfo.User),
		IconUrl:     "http://gdrive2slack.optionfactory.net/gdrive2slack.png",
		Attachments: attachments,
	}
}

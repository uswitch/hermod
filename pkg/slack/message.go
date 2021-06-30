package slack

import (
	"github.com/slack-go/slack"
)

func (c *Client) SendMessage(message string) error {
	_, _, err := c.client.PostMessage(
		c.channelID,
		slack.MsgOptionText(message, false),
	)

	return err
}

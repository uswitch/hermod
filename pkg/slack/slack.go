package slack

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// color codes for slack message
const (
	OrangeColor = "#e69500" // Rollout Started
	GreenColor  = "#36a64f" // Rollout Successful
	RedColor    = "#e60000" // Rollout Failed
)

// This package will consists of code which will make slack API call to post a message
type Client struct {
	client *slack.Client
}

// NewClient instantiates the expected Slack Client struct fields
func NewClient() (*Client, error) {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("SLACK_TOKEN environment variable not set")
	}

	return &Client{client: slack.New(token)}, nil
}

func (c *Client) SendMessage(channel, message, color string) error {
	log.Debugf("sending alert \"%s\" to '%s'", message, channel)

	_, _, err := c.client.PostMessage(channel, slack.MsgOptionAttachments(createSlackAttachment(message, color)))

	if err != nil {
		return fmt.Errorf("failed to send message \"%s\" to '%s': %s", message, channel, err)
	}

	return err
}

func createSlackAttachment(msg, color string) slack.Attachment {

	attachment := slack.Attachment{
		Footer: "hermod",
		Fields: []slack.AttachmentField{
			{
				Value: msg,
			},
		},
	}

	attachment.Color = color

	attachment.MarkdownIn = []string{"fields"}

	return attachment
}

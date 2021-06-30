package slack

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

// This package will consists of code which will make slack API call to post a message
type Client struct {
	client    *slack.Client
	channelID string
}

// NewClient instantiates the expected Slack Client struct fields
func NewClient(channelID string) (*Client, error) {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("SLACK_TOKEN environment variable not set")
	}

	return &Client{client: slack.New(token), channelID: channelID}, nil
}

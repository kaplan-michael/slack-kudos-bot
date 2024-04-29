package commands

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"regexp"
	"strconv"
	"strings"
)

func NewKudosHandler() *RegexCommandHandler {
	return &RegexCommandHandler{
		Pattern:    regexp.MustCompile(`/kudos`),
		HandleFunc: KudosCommand,
	}
}

// KudosCommand handles the "/kudos" slash command.
func KudosCommand(client *socketmode.Client, evt *socketmode.Event) error {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		log.Warnf("expected SlashCommand in event data")
		return nil
	}

	// Default to showing top 5 users if no number is specified
	topCount := 5
	args := strings.Fields(cmd.Text)
	if len(args) > 0 {
		var err error
		topCount, err = strconv.Atoi(args[0])
		if err != nil {
			msg := "Invalid number specified. Please enter a valid number."
			_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(msg, false))
			if err != nil {
				return fmt.Errorf("failed to post message: %v", err)
			}
			return fmt.Errorf("invalid number specified: %v", err)
		}
	}

	users, err := GetTopKudosUsers(topCount)
	if err != nil {
		msg := "Failed to retrieve top kudos users."
		_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(msg, false))
		if err != nil {
			return fmt.Errorf("failed to post message: %v", err)
		}
		return fmt.Errorf("failed to retrieve top kudos users: %v", err)
	}

	response := fmt.Sprintf("Top %d kudos users:\n", topCount)
	for _, user := range users {
		response += fmt.Sprintf("<@%s> - %d kudos\n", user.UserID, user.Count)
	}

	_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(response, false))
	if err != nil {
		return fmt.Errorf("failed to post message: %v", err)
	}
	return nil
}

// GetTopKudosUsers retrieves the top 'limit' users with the most kudos.
func GetTopKudosUsers(limit int) ([]KudosUser, error) {
	var users []KudosUser
	rows, err := database.DB.Query(`
        SELECT user_id, count 
        FROM kudos 
        ORDER BY count DESC 
        LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top kudos users: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user KudosUser
		if err := rows.Scan(&user.UserID, &user.Count); err != nil {
			return nil, fmt.Errorf("failed to scan kudos user: %v", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	return users, nil
}

// KudosUser struct to hold user ID and kudos count.
type KudosUser struct {
	UserID string
	Count  int
}

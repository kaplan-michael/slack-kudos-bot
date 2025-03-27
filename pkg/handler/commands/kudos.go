package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
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

	// Get the team ID from the slash command
	teamID := cmd.TeamID
	if teamID == "" {
		log.Warn("Could not determine team ID for kudos command")
		return fmt.Errorf("could not determine team ID")
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

	users, err := GetTopKudosUsers(teamID, topCount)
	if err != nil {
		// Check for workspace not found error specifically
		if strings.Contains(err.Error(), "workspace not found") {
			msg := "This workspace hasn't been set up yet. Make sure the OAuth installation has been completed."
			_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(msg, false))
			if err != nil {
				return fmt.Errorf("failed to post message: %v", err)
			}
			return nil
		}

		// Other errors
		msg := "Failed to retrieve top kudos users."
		_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(msg, false))
		if err != nil {
			return fmt.Errorf("failed to post message: %v", err)
		}
		return fmt.Errorf("failed to retrieve top kudos users: %v", err)
	}

	// Check if any users were found
	if len(users) == 0 {
		response := "No kudos have been given in this workspace yet. Be the first to give kudos by mentioning someone with `++`!"
		_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(response, false))
		if err != nil {
			return fmt.Errorf("failed to post message: %v", err)
		}
		return nil
	}

	// Build the response with the top users
	response := fmt.Sprintf("Top %d kudos users in this workspace:\n", topCount)
	for _, user := range users {
		response += fmt.Sprintf("<@%s> - %d kudos\n", user.UserID, user.Count)
	}

	_, _, err = client.PostMessage(cmd.ChannelID, slack.MsgOptionText(response, false))
	if err != nil {
		return fmt.Errorf("failed to post message: %v", err)
	}
	return nil
}

// Note: This function is no longer used as we get the team ID directly
// from the API or command data

// GetTopKudosUsers retrieves the top 'limit' users with the most kudos for a specific workspace.
func GetTopKudosUsers(teamID string, limit int) ([]KudosUser, error) {
	var users []KudosUser

	// First check if workspace exists
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM workspaces WHERE team_id = ?`, teamID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("error checking workspace: %w", err)
	}

	// If the workspace doesn't exist in our database, we can't get kudos yet
	if count == 0 {
		return nil, fmt.Errorf("workspace %s not found in database", teamID)
	}

	// Query workspace_kudos table for multi-tenant support
	rows, err := database.DB.Query(`
        SELECT user_id, count 
        FROM workspace_kudos 
        WHERE team_id = ?
        ORDER BY count DESC 
        LIMIT ?`, teamID, limit)
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

	// If no users found, return empty slice
	if len(users) == 0 {
		return []KudosUser{}, nil
	}

	return users, nil
}

// KudosUser struct to hold user ID and kudos count.
type KudosUser struct {
	UserID string
	Count  int
}

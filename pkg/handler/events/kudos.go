package events

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func NewKudosHandler() *RegexMessageHandler {
	return &RegexMessageHandler{
		Pattern:    regexp.MustCompile(`<@(\w+)>\s*\+\+`),
		HandleFunc: handleKudos,
	}
}

// handleKudos processes messages that give kudos to users.
func handleKudos(client *socketmode.Client, msgEvent *slackevents.MessageEvent) error {
	// Get team ID using auth test
	api := client.Client
	authInfo, err := api.AuthTest()
	if err != nil {
		log.Warnf("Error getting team ID from auth test: %v", err)
		return fmt.Errorf("could not get team info: %w", err)
	}

	teamID := authInfo.TeamID
	if teamID == "" {
		log.Warn("Could not determine team ID for kudos event, skipping")
		return fmt.Errorf("empty team ID from auth test")
	}

	log.Infof("Using team ID: %s", teamID)

	userID := extractUserID(msgEvent.Text)
	if userID == "" {
		return fmt.Errorf("could not extract user ID from message")
	}

	log.Infof("User %s in workspace %s received kudos", userID, teamID)

	count, err := incrementKudosCount(teamID, userID)
	if err != nil {
		return fmt.Errorf("failed to increment kudos for user %s in workspace %s: %v", userID, teamID, err)
	}

	response := fmt.Sprintf("<@%s> got a kudos! 🎉\n Now has %d kudos in this workspace!", userID, count)
	_, _, err = client.PostMessage(msgEvent.Channel, slack.MsgOptionText(response, false))
	return err
}

// extractUserID extracts a user ID from the message text.
func extractUserID(text string) string {
	re := regexp.MustCompile(`<@(\w+)>`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Note: This function is no longer used as we get the team ID directly
// from the API or command data

// incrementKudosCount increments the kudos count for the given user and returns the new count.
func incrementKudosCount(teamID, userID string) (int, error) {
	var newCount int

	// First check if workspace exists (to avoid foreign key constraint error)
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM workspaces WHERE team_id = ?`, teamID).Scan(&count)
	if err != nil {
		log.Warnf("Error checking if workspace exists: %v", err)
		return 0, fmt.Errorf("error checking workspace: %w", err)
	}

	// If the workspace doesn't exist in our database, we can't record kudos yet
	if count == 0 {
		log.Warnf("Workspace %s not found in database. Make sure OAuth setup is complete.", teamID)
		return 0, fmt.Errorf("workspace %s not found in database", teamID)
	}

	// Using workspace_kudos table for multi-tenant support
	err = database.DB.QueryRow(`
        INSERT INTO workspace_kudos (team_id, user_id, count) 
        VALUES (?, ?, 1) 
        ON CONFLICT(team_id, user_id) 
        DO UPDATE SET count = count + 1 
        WHERE team_id = ? AND user_id = ?
        RETURNING count;`, teamID, userID, teamID, userID).Scan(&newCount)

	if err != nil {
		log.Warnf("Failed to increment and fetch kudos count for user %s in workspace %s: %v", userID, teamID, err)
		return 0, err
	}

	log.Infof("Incremented kudos count for user %s in workspace %s, now has %d", userID, teamID, newCount)
	return newCount, nil
}

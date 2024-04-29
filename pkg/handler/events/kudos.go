package events

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"regexp"
)

func NewKudosHandler() *RegexMessageHandler {
	return &RegexMessageHandler{
		Pattern:    regexp.MustCompile(`<@(\w+)>\s*\+\+`),
		HandleFunc: handleKudos,
	}
}

// handleKudos processes messages that give kudos to users.
func handleKudos(client *socketmode.Client, msgEvent *slackevents.MessageEvent) error {
	userID := extractUserID(msgEvent.Text)
	if userID == "" {
		return fmt.Errorf("could not extract user ID from message")
	}

	count, err := incrementKudosCount(userID)
	if err != nil {
		return fmt.Errorf("failed to increment kudos for user %s: %v", userID, err)
	}

	response := fmt.Sprintf("<@%s> got a kudos! 🎉\n Now has %d kudos!", userID, count)
	_, _, err = client.PostMessage(msgEvent.Channel, slack.MsgOptionText(response, false))
	return err
}

// extractUserID extracts a user ID from the message text.
func extractUserID(text string) string {
	re := regexp.MustCompile(`@(\w+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// incrementKudosCount increments the kudos count for the given user and returns the new count.
func incrementKudosCount(userID string) (int, error) {
	var newCount int
	// Using a transaction to ensure the operation is atomic
	err := database.DB.QueryRow(`
        INSERT INTO kudos (user_id, count) 
        VALUES (?, 1) 
        ON CONFLICT(user_id) 
        DO UPDATE SET count = count + 1 
        WHERE user_id = ?
        RETURNING count;`, userID, userID).Scan(&newCount)
	log.Infof("Incrementing kudos count for user %s, now has %d", userID, newCount)

	if err != nil {
		log.Warnf("failed to increment and fetch kudos count for user %s: %v", userID, err)
		return 0, nil
	}

	return newCount, nil
}

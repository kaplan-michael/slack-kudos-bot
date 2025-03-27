package oauth2

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/kaplan-michael/slack-kudos/pkg/config"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/slack-go/slack"
	"net/http"
	"strings"
	"time"
)

// WorkspaceCredentials represents the OAuth tokens for a Slack workspace
type WorkspaceCredentials struct {
	TeamID        string    `json:"team_id"`
	TeamName      string    `json:"team_name"`
	AccessToken   string    `json:"access_token"`
	BotUserID     string    `json:"bot_user_id"`
	Scopes        string    `json:"scopes"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	RefreshToken  string    `json:"refresh_token,omitempty"`
	LastUpdated   time.Time `json:"last_updated"`
}

// OAuthHandler handles the OAuth flow endpoints
type OAuthHandler struct {
	clientID      string
	clientSecret  string
	redirectURI   string
	scopes        []string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{
		clientID:     config.AppConfig.SlackClientID,
		clientSecret: config.AppConfig.SlackClientSecret,
		redirectURI:  config.AppConfig.SlackRedirectURI,
		scopes: []string{
			"channels:history", 
			"channels:read", 
			"chat:write", 
			"commands",
			"groups:history", 
			"im:history", 
			"users:read",
		},
	}
}

// StartOAuth initiates the OAuth process by redirecting to Slack's authorization page
func (h *OAuthHandler) StartOAuth(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		h.clientID,
		strings.Join(h.scopes, ","),
		h.redirectURI,
	)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// OAuthCallback handles the OAuth callback from Slack
func (h *OAuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	// Create HTTP client
	httpClient := &http.Client{}

	// Exchange the code for an access token
	response, err := slack.GetOAuthV2Response(
		httpClient,
		h.clientID,
		h.clientSecret,
		code,
		h.redirectURI,
	)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store the workspace information in the database
	creds := WorkspaceCredentials{
		TeamID:        response.Team.ID,
		TeamName:      response.Team.Name,
		AccessToken:   response.AccessToken,
		BotUserID:     response.BotUserID,
		Scopes:        response.Scope,
		LastUpdated:   time.Now(),
	}

	// If token is refreshable, store refresh token and expiry
	if response.RefreshToken != "" {
		creds.RefreshToken = response.RefreshToken
		expiresIn := time.Duration(response.ExpiresIn) * time.Second
		creds.ExpiresAt = time.Now().Add(expiresIn)
	}

	// Save workspace credentials
	if err := SaveWorkspaceCredentials(creds); err != nil {
		http.Error(w, "Failed to save workspace credentials: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get bot info to display username
	api := slack.New(creds.AccessToken)
	botInfo, err := api.AuthTest()
	botName := "the bot"
	if err == nil && botInfo != nil {
		botName = "@" + botInfo.User
	}

	// Display success page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<html>
			<head>
				<title>Slack Kudos App</title>
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
						line-height: 1.6;
						color: #333;
						max-width: 800px;
						margin: 0 auto;
						padding: 20px;
					}
					h1 { color: #1264A3; }
					h2 { color: #4A154B; margin-top: 30px; }
					.container {
						background: #f9f9f9;
						border-radius: 10px;
						padding: 30px;
						box-shadow: 0 4px 6px rgba(0,0,0,0.1);
					}
					.steps {
						margin-top: 20px;
						margin-bottom: 20px;
					}
					.step {
						margin-bottom: 15px;
					}
				</style>
			</head>
			<body>
				<div class="container">
					<h1>Installation Successful! 🎉</h1>
					<p>Kudos bot has been successfully installed to your workspace: <strong>%s</strong></p>
					
					<h2>Next Steps:</h2>
					<div class="steps">
						<div class="step">
							<strong>1. Invite %s to channels</strong> where you want to use it:
							<pre>    /invite %s</pre>
						</div>
						<div class="step">
							<strong>2. Give kudos</strong> by mentioning someone with ++:
							<pre>    @user ++</pre>
						</div>
						<div class="step">
							<strong>3. Check the leaderboard</strong> with the slash command:
							<pre>    /kudos</pre>
						</div>
					</div>
					
					<p><em>Note: The bot must be in a channel to detect kudos mentions and respond to commands.</em></p>
				</div>
			</body>
		</html>
	`, creds.TeamName, botName, botName)
}

// SaveWorkspaceCredentials saves the workspace credentials to the database
func SaveWorkspaceCredentials(creds WorkspaceCredentials) error {
	scopesJSON, err := json.Marshal(creds.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO workspaces (
			team_id, team_name, access_token, bot_user_id, 
			scopes, expires_at, refresh_token, last_updated
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAt *time.Time
	if !creds.ExpiresAt.IsZero() {
		expiresAt = &creds.ExpiresAt
	}

	_, err = database.DB.Exec(
		query,
		creds.TeamID,
		creds.TeamName,
		creds.AccessToken,
		creds.BotUserID,
		string(scopesJSON),
		expiresAt,
		creds.RefreshToken,
		time.Now(),
	)
	
	return err
}

// GetWorkspaceCredentials retrieves credentials for a specific workspace
func GetWorkspaceCredentials(teamID string) (WorkspaceCredentials, error) {
	var creds WorkspaceCredentials
	var expiresAt sql.NullTime
	var scopesStr string

	query := `
		SELECT team_id, team_name, access_token, bot_user_id, 
		       scopes, expires_at, refresh_token, last_updated 
		FROM workspaces 
		WHERE team_id = ?
	`
	
	err := database.DB.QueryRow(query, teamID).Scan(
		&creds.TeamID,
		&creds.TeamName,
		&creds.AccessToken,
		&creds.BotUserID,
		&scopesStr,
		&expiresAt,
		&creds.RefreshToken,
		&creds.LastUpdated,
	)
	
	if err != nil {
		return creds, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Process expires_at if it's valid
	if expiresAt.Valid {
		creds.ExpiresAt = expiresAt.Time
	}

	// Parse scopes
	creds.Scopes = scopesStr

	return creds, nil
}

// GetAllWorkspaceCredentials retrieves all workspace credentials
func GetAllWorkspaceCredentials() ([]WorkspaceCredentials, error) {
	var workspaces []WorkspaceCredentials

	query := `
		SELECT team_id, team_name, access_token, bot_user_id, 
		       scopes, expires_at, refresh_token, last_updated 
		FROM workspaces
	`
	
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspaces: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var creds WorkspaceCredentials
		var expiresAt sql.NullTime
		var scopesStr string

		err := rows.Scan(
			&creds.TeamID,
			&creds.TeamName,
			&creds.AccessToken,
			&creds.BotUserID,
			&scopesStr,
			&expiresAt,
			&creds.RefreshToken,
			&creds.LastUpdated,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace row: %w", err)
		}

		// Process expires_at if it's valid
		if expiresAt.Valid {
			creds.ExpiresAt = expiresAt.Time
		}

		// Parse scopes
		creds.Scopes = scopesStr

		workspaces = append(workspaces, creds)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspace rows: %w", err)
	}

	return workspaces, nil
}

// RefreshTokenIfNeeded checks if a token needs refreshing and refreshes it
// This is a package-level function to be called from main
func RefreshTokenIfNeeded(teamID string) error {
	creds, err := GetWorkspaceCredentials(teamID)
	if err != nil {
		return err
	}

	// If token doesn't expire or is not close to expiry, return
	if creds.ExpiresAt.IsZero() || time.Until(creds.ExpiresAt) > 1*time.Hour {
		return nil
	}

	// Use client ID and secret from config
	clientID := config.AppConfig.SlackClientID
	clientSecret := config.AppConfig.SlackClientSecret

	// Create HTTP client
	httpClient := &http.Client{}

	// Token needs refreshing
	resp, err := slack.RefreshOAuthV2Token(httpClient, creds.RefreshToken, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update credentials with new tokens
	creds.AccessToken = resp.AccessToken
	creds.ExpiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	creds.LastUpdated = time.Now()
	
	if resp.RefreshToken != "" {
		creds.RefreshToken = resp.RefreshToken
	}

	// Save updated credentials
	return SaveWorkspaceCredentials(creds)
}
package main

import (
	"context"
	"database/sql"
	"fmt"
	l "log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/kaplan-michael/slack-kudos/pkg/config"
	"github.com/kaplan-michael/slack-kudos/pkg/database"
	"github.com/kaplan-michael/slack-kudos/pkg/dispatcher"
	"github.com/kaplan-michael/slack-kudos/pkg/oauth2"
	"github.com/kaplan-michael/slack-kudos/pkg/utils"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// WorkspaceClient holds the socket mode client for a workspace
type WorkspaceClient struct {
	TeamID string
	Client *socketmode.Client
	API    *slack.Client
}

// WorkspaceManager manages all workspace clients
type WorkspaceManager struct {
	dispatcher *dispatcher.Dispatcher
	clients    map[string]*WorkspaceClient
	mu         sync.RWMutex
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager(disp *dispatcher.Dispatcher) *WorkspaceManager {
	return &WorkspaceManager{
		dispatcher: disp,
		clients:    make(map[string]*WorkspaceClient),
		mu:         sync.RWMutex{},
	}
}

// AddWorkspace adds a new workspace client
func (wm *WorkspaceManager) AddWorkspace(creds oauth2.WorkspaceCredentials) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check if we already have this workspace
	if _, exists := wm.clients[creds.TeamID]; exists {
		// Create a new API client with the refreshed token
		wm.clients[creds.TeamID].API = slack.New(creds.AccessToken)
		return nil
	}

	// Create a new Slack API client for this workspace
	api := slack.New(
		creds.AccessToken,
		slack.OptionDebug(config.AppConfig.Debug),
		// Socket Mode requires app-level token (same for all workspaces)
		slack.OptionAppLevelToken(config.AppConfig.SlackAppToken),
	)

	// Create a new Socket Mode client
	client := socketmode.New(
		api,
		socketmode.OptionDebug(config.AppConfig.Debug),
		socketmode.OptionLog(l.New(os.Stdout, fmt.Sprintf("workspace-%s: ", creds.TeamID), l.Lshortfile|l.LstdFlags)),
	)

	// Create workspace client
	wsClient := &WorkspaceClient{
		TeamID: creds.TeamID,
		Client: client,
		API:    api,
	}

	// Store the client
	wm.clients[creds.TeamID] = wsClient

	// Start listening for events
	go func() {
		for evt := range client.Events {
			// We can't modify the Request.Context directly as it doesn't exist
			// Instead, we'll store team ID in a workspace map in memory
			// and look it up when needed from the API endpoints

			// Dispatch events
			if err := wm.dispatcher.Dispatch(&evt, client); err != nil {
				log.Warnf("Error processing event for workspace %s: %s\n", creds.TeamID, err)
			}
		}
	}()

	// Start the client
	go func() {
		log.Infof("Starting Socket Mode client for workspace: %s (%s)", creds.TeamName, creds.TeamID)
		if err := client.Run(); err != nil {
			log.Warnf("Socket Mode client for workspace %s stopped: %v", creds.TeamID, err)
		}
	}()

	log.Infof("Added workspace: %s (%s)", creds.TeamName, creds.TeamID)
	return nil
}

// RemoveWorkspace removes a workspace client
func (wm *WorkspaceManager) RemoveWorkspace(teamID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.clients[teamID]; exists {
		// Just remove it from the map - socket mode client will be garbage collected
		delete(wm.clients, teamID)
		log.Infof("Removed workspace: %s", teamID)
	}
}

// GetWorkspaceClient gets a workspace client by team ID
func (wm *WorkspaceManager) GetWorkspaceClient(teamID string) (*WorkspaceClient, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	client, exists := wm.clients[teamID]
	return client, exists
}

func main() {
	// Initialize configuration
	config.Init()

	// Initialize database
	err := database.InitDB()
	if err != nil {
		log.Fatalf("Error initializing database: %s\n", err)
	}

	// Initialize the central dispatcher
	disp := dispatcher.NewDispatcher()

	// Create workspace manager for multi-tenant support
	workspaceManager := NewWorkspaceManager(disp)

	// Create HTTP server for OAuth flow
	oauthHandler := oauth2.NewOAuthHandler()

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/start", oauthHandler.StartOAuth)
	mux.HandleFunc("/oauth/callback", oauthHandler.OAuthCallback)

	// Create public landing page and installation instructions
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
						h1 {
							color: #1264A3; /* Slack blue */
						}
						.container {
							background: #f9f9f9;
							border-radius: 10px;
							padding: 30px;
							box-shadow: 0 4px 6px rgba(0,0,0,0.1);
						}
						.btn {
							display: inline-block;
							background-color: #4A154B; /* Slack purple */
							color: white;
							padding: 12px 24px;
							text-decoration: none;
							border-radius: 4px;
							font-weight: bold;
							margin-top: 20px;
						}
						.features {
							margin-top: 30px;
						}
						.feature {
							margin-bottom: 15px;
						}
					</style>
				</head>
				<body>
					<div class="container">
						<h1>Slack Kudos App</h1>
						<p>A simple way to give recognition to your team members in Slack!</p>
						
						<div class="features">
							<div class="feature">
								<strong>Give kudos</strong> - Mention a user with "++" to give them kudos (e.g., "@user ++")
							</div>
							<div class="feature">
								<strong>View leaderboard</strong> - Use the "/kudos" command to see who's received the most kudos
							</div>
							<div class="feature">
								<strong>Easy to use</strong> - No configuration needed, just install and start recognizing your teammates
							</div>
						</div>
						
						<a href="/oauth/start" class="btn">Install on Slack</a>
					</div>
				</body>
			</html>
		`)
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.AppConfig.ServerPort),
		Handler: mux,
	}

	// Start HTTP/HTTPS server in a goroutine
	go func() {
		if config.AppConfig.Debug {
			// In debug mode, generate self-signed certificate and use HTTPS
			log.Info("Running in debug mode with self-signed certificate")
			certPath, keyPath, err := utils.GenerateSelfSignedCert(config.AppConfig.BaseURL)
			if err != nil {
				log.Fatalf("Failed to generate self-signed certificate: %v", err)
			}

			log.Info("=========================================================")
			log.Infof("HTTPS server listening on port %d", config.AppConfig.ServerPort)
			log.Infof("Base URL: %s", config.AppConfig.BaseURL)
			log.Infof("Redirect URI: %s", config.AppConfig.SlackRedirectURI)
			log.Info("=========================================================")
			log.Info("⚠️  IMPORTANT: Since this is a self-signed certificate, you'll need to:")
			log.Infof("   1. Open %s in your browser", config.AppConfig.BaseURL)
			log.Info("   2. Click 'Advanced' and then 'Proceed anyway' to accept the certificate")
			log.Info("   3. Then try the Slack OAuth flow again")
			log.Info("=========================================================")

			if err := srv.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Error starting HTTPS server: %v", err)
			}
		} else {
			// In production mode, use HTTP (assuming reverse proxy for HTTPS)
			log.Info("=========================================================")
			log.Infof("HTTP server listening on port %d", config.AppConfig.ServerPort)
			log.Infof("Base URL: %s", config.AppConfig.BaseURL)
			log.Infof("Redirect URI: %s", config.AppConfig.SlackRedirectURI)
			log.Info("=========================================================")

			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Error starting HTTP server: %v", err)
			}
		}
	}()

	// Load all workspaces from database
	workspaces, err := oauth2.GetAllWorkspaceCredentials()
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("Error loading workspaces: %v", err)
	}

	// Add each workspace
	for _, workspace := range workspaces {
		// Skip adding workspaces if there are issues with tokens
		if workspace.AccessToken == "" {
			log.Warnf("Skipping workspace %s due to missing access token", workspace.TeamID)
			continue
		}

		// Log token info for debugging
		if config.AppConfig.Debug {
			log.Infof("Workspace %s (%s) token: %s...",
				workspace.TeamName,
				workspace.TeamID,
				workspace.AccessToken[:10]+"..."+workspace.AccessToken[len(workspace.AccessToken)-5:])
		}

		// Check if token needs refreshing
		if err := oauth2.RefreshTokenIfNeeded(workspace.TeamID); err != nil {
			log.Warnf("Failed to refresh token for workspace %s: %v", workspace.TeamID, err)
			continue
		}

		// Get refreshed credentials
		refreshedCreds, err := oauth2.GetWorkspaceCredentials(workspace.TeamID)
		if err != nil {
			log.Warnf("Failed to get refreshed credentials for workspace %s: %v", workspace.TeamID, err)
			continue
		}

		// Add the workspace
		if err := workspaceManager.AddWorkspace(refreshedCreds); err != nil {
			log.Warnf("Failed to add workspace %s: %v", workspace.TeamID, err)
		} else {
			// Get bot info to display name for inviting to channels
			api := slack.New(refreshedCreds.AccessToken)
			botInfo, err := api.AuthTest()
			if err != nil {
				log.Warnf("Failed to get bot info for workspace %s: %v", workspace.TeamID, err)
			} else {
				log.Infof("✅ Bot installed to workspace %s as @%s", refreshedCreds.TeamName, botInfo.User)
				log.Infof("   👉 Please invite @%s to channels where you want to use it", botInfo.User)
			}
		}
	}

	log.Infof("Bot is running with %d workspaces...", len(workspaces))

	// Register webhook handler for Slack events if needed
	mux.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
		// Handle Slack events API requests
		// This is needed if you want to use Events API instead of Socket Mode
		w.WriteHeader(http.StatusOK)
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}

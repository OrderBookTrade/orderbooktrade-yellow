package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"orderbook-backend/internal/api"
	"orderbook-backend/internal/config"
	"orderbook-backend/internal/engine"
	"orderbook-backend/internal/market"
	"orderbook-backend/internal/yellow"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Orderbook Backend (Prediction Market Mode)...")

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize market orderbooks (separate YES/NO orderbooks per market)
	marketOrderbooks := engine.NewMarketOrderbooks()
	log.Println("Market orderbooks initialized")

	// Initialize market manager (prediction markets)
	marketManager := market.NewManager()
	lifecycleManager := market.NewLifecycleManager(marketManager)
	log.Println("Market manager initialized")

	// Initialize position manager
	positions := engine.NewPositionManager()
	log.Println("Position manager initialized")

	// Initialize Yellow Network client (optional - only if private key is set)
	var yellowClient *yellow.Client
	var sessions *yellow.SessionManager

	log.Println("Initializing Yellow SDK...")
	if cfg.PrivateKey != "" {
		signer, err := yellow.NewSigner(cfg.PrivateKey)
		if err != nil {
			log.Printf("❌ Yellow SDK: Failed to initialize signer: %v", err)
		} else {
			log.Printf("✓ Yellow SDK: Signer initialized (address: %s)", signer.Address().Hex())
			yellowClient = yellow.NewClient(cfg.YellowNodeURL, signer)

			// Connect to Yellow Network
			log.Printf("  Connecting to Yellow Network: %s", cfg.YellowNodeURL)
			ctx := context.Background()
			if err := yellowClient.Connect(ctx); err != nil {
				log.Printf("❌ Yellow SDK: Connection failed: %v", err)
			} else {
				log.Println("✓ Yellow SDK: WebSocket connected")
				// Authenticate
				if err := yellowClient.Authenticate(ctx); err != nil {
					log.Printf("❌ Yellow SDK: Authentication failed: %v", err)
				} else {
					sessions = yellow.NewSessionManager(yellowClient, signer)
					log.Println("✓ Yellow SDK: Authenticated successfully")
					log.Printf("🟢 Yellow Network: CONNECTED and ready")
				}
			}
		}
	} else {
		log.Println("⚪ Yellow SDK: Disabled (no PRIVATE_KEY set)")
	}

	// Initialize API server
	server := api.NewServer(cfg, marketOrderbooks, yellowClient, sessions, marketManager, positions)

	// Start lifecycle manager (auto-lock markets when resolution time passes)
	ctx, cancel := context.WithCancel(context.Background())
	lifecycleManager.Start(ctx)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		cancel()
		lifecycleManager.Stop()
		if yellowClient != nil {
			yellowClient.Close()
		}
		os.Exit(0)
	}()

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

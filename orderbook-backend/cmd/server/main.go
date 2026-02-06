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
	"orderbook-backend/internal/yellow"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Orderbook Backend...")

	// Load configuration
	cfg := config.Load()

	// Initialize matching engine
	orderbook := engine.NewOrderbook()
	log.Println("Matching engine initialized")

	// Initialize Yellow Network client (optional - only if private key is set)
	var yellowClient *yellow.Client
	var sessions *yellow.SessionManager

	if cfg.PrivateKey != "" {
		signer, err := yellow.NewSigner(cfg.PrivateKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize signer: %v", err)
		} else {
			yellowClient = yellow.NewClient(cfg.YellowNodeURL, signer)

			// Connect to Yellow Network
			ctx := context.Background()
			if err := yellowClient.Connect(ctx); err != nil {
				log.Printf("Warning: Failed to connect to Yellow Network: %v", err)
			} else {
				// Authenticate
				if err := yellowClient.Authenticate(ctx); err != nil {
					log.Printf("Warning: Failed to authenticate with Yellow Network: %v", err)
				} else {
					sessions = yellow.NewSessionManager(yellowClient)
					log.Println("Connected to Yellow Network")
				}
			}
		}
	} else {
		log.Println("Yellow Network integration disabled (no PRIVATE_KEY set)")
	}

	// Initialize API server
	server := api.NewServer(cfg, orderbook, yellowClient, sessions)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
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

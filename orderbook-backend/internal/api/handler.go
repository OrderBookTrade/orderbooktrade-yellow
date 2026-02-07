package api

import (
	"log"
	"net/http"

	"orderbook-backend/internal/config"
	"orderbook-backend/internal/engine"
	"orderbook-backend/internal/market"
	"orderbook-backend/internal/state"
	"orderbook-backend/internal/yellow"
)

// Server holds all dependencies for the HTTP server
type Server struct {
	cfg              *config.Config
	marketOrderbooks *engine.MarketOrderbooks
	yellowClient     *yellow.Client
	sessions         *yellow.SessionManager
	allocations      *state.Allocations
	wsHub            *Hub
	marketManager    *market.Manager
	positions        *engine.PositionManager
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	marketOrderbooks *engine.MarketOrderbooks,
	yellowClient *yellow.Client,
	sessions *yellow.SessionManager,
	marketManager *market.Manager,
	positions *engine.PositionManager,
) *Server {
	return &Server{
		cfg:              cfg,
		marketOrderbooks: marketOrderbooks,
		yellowClient:     yellowClient,
		sessions:         sessions,
		wsHub:            NewHub(),
		marketManager:    marketManager,
		positions:        positions,
	}
}

// SetAllocations sets the allocations tracker
func (s *Server) SetAllocations(alloc *state.Allocations) {
	s.allocations = alloc
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Market endpoints (prediction market)
	mux.HandleFunc("POST /api/market", s.handleCreateMarket)
	mux.HandleFunc("GET /api/markets", s.handleListMarkets)
	mux.HandleFunc("GET /api/market/{id}", s.handleGetMarket)
	mux.HandleFunc("POST /api/market/{id}/resolve", s.handleResolveMarket)

	// Order endpoints
	mux.HandleFunc("POST /api/order", s.handlePlaceOrder)
	mux.HandleFunc("GET /api/orderbook", s.handleGetOrderbook)
	mux.HandleFunc("DELETE /api/order/{id}", s.handleCancelOrder)
	mux.HandleFunc("GET /api/trades", s.handleGetTrades)

	// Position endpoints
	mux.HandleFunc("GET /api/position/{userId}", s.handleGetPosition)
	mux.HandleFunc("POST /api/deposit", s.handleDeposit)
	mux.HandleFunc("POST /api/mint", s.handleMintShares)

	// Session endpoints
	mux.HandleFunc("POST /api/session", s.handleCreateSession)
	mux.HandleFunc("DELETE /api/session/{id}", s.handleCloseSession)

	// Settlement endpoint
	mux.HandleFunc("POST /api/settle", s.handleSettle)

	// WebSocket endpoint
	mux.HandleFunc("GET /ws", s.handleWebSocket)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Trade callbacks are set per-market when markets are created

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	// Add CORS middleware
	handler := corsMiddleware(mux)

	addr := ":" + s.cfg.ServerPort
	log.Printf("Server starting on %s", addr)
	return http.ListenAndServe(addr, handler)
}

// handleHealth is the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

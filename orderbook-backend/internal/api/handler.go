package api

import (
	"log"
	"net/http"

	"orderbook-backend/internal/config"
	"orderbook-backend/internal/engine"
	"orderbook-backend/internal/state"
	"orderbook-backend/internal/yellow"
)

// Server holds all dependencies for the HTTP server
type Server struct {
	cfg          *config.Config
	orderbook    *engine.Orderbook
	yellowClient *yellow.Client
	sessions     *yellow.SessionManager
	allocations  *state.Allocations
	wsHub        *Hub
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	orderbook *engine.Orderbook,
	yellowClient *yellow.Client,
	sessions *yellow.SessionManager,
) *Server {
	return &Server{
		cfg:          cfg,
		orderbook:    orderbook,
		yellowClient: yellowClient,
		sessions:     sessions,
		wsHub:        NewHub(),
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

	// Order endpoints
	mux.HandleFunc("POST /api/order", s.handlePlaceOrder)
	mux.HandleFunc("GET /api/orderbook", s.handleGetOrderbook)
	mux.HandleFunc("DELETE /api/order/{id}", s.handleCancelOrder)
	mux.HandleFunc("GET /api/trades", s.handleGetTrades)

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

	// Set up trade notifications to broadcast
	s.orderbook.SetTradeCallback(func(trade *engine.Trade) {
		s.wsHub.Broadcast(Message{
			Type: "trade",
			Data: trade,
		})
	})

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

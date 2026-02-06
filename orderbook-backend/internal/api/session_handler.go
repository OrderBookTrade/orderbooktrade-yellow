package api

import (
	"encoding/json"
	"net/http"

	"orderbook-backend/internal/yellow"
)

// CreateSessionRequest is the request body for creating a session
type CreateSessionRequest struct {
	Participants []string            `json:"participants"`
	Allocations  []yellow.Allocation `json:"allocations"`
}

// CreateSessionResponse is the response for a created session
type CreateSessionResponse struct {
	ChannelID string `json:"channel_id"`
	Status    string `json:"status"`
}

// handleCreateSession handles POST /api/session
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "session manager not initialized")
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Participants) < 2 {
		writeError(w, http.StatusBadRequest, "at least 2 participants required")
		return
	}

	session, err := s.sessions.CreateSession(
		r.Context(),
		req.Participants,
		req.Allocations,
		s.cfg.AdjudicatorAddr,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, CreateSessionResponse{
		ChannelID: session.GetChannelID(),
		Status:    "created",
	})
}

// handleCloseSession handles DELETE /api/session/{id}
func (s *Server) handleCloseSession(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "session manager not initialized")
		return
	}

	channelID := r.PathValue("id")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "session id required")
		return
	}

	if err := s.sessions.CloseSession(r.Context(), channelID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "closed",
		"channel_id": channelID,
	})
}

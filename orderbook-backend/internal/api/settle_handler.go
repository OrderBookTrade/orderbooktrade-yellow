package api

import (
	"encoding/json"
	"net/http"
)

// SettleRequest is the request body for settlement
type SettleRequest struct {
	ChannelID string `json:"channel_id"`
	Type      string `json:"type"` // "cooperative" or "dispute"
}

// SettleResponse is the response for settlement
type SettleResponse struct {
	Status    string `json:"status"`
	ChannelID string `json:"channel_id"`
	TxHash    string `json:"tx_hash,omitempty"`
}

// handleSettle handles POST /api/settle
func (s *Server) handleSettle(w http.ResponseWriter, r *http.Request) {
	var req SettleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ChannelID == "" {
		writeError(w, http.StatusBadRequest, "channel_id required")
		return
	}

	switch req.Type {
	case "cooperative":
		// In cooperative close, we close the session normally
		// The Yellow Network handles the on-chain settlement
		if s.sessions != nil {
			if err := s.sessions.CloseSession(r.Context(), req.ChannelID); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		writeJSON(w, http.StatusOK, SettleResponse{
			Status:    "settled",
			ChannelID: req.ChannelID,
		})

	case "dispute":
		// In dispute mode, we would need to:
		// 1. Collect the latest signed state
		// 2. Submit it to the on-chain adjudicator contract
		// This requires an Ethereum client connection

		// For now, return a placeholder response
		writeJSON(w, http.StatusOK, SettleResponse{
			Status:    "dispute_initiated",
			ChannelID: req.ChannelID,
			TxHash:    "", // Would be the actual tx hash
		})

	default:
		writeError(w, http.StatusBadRequest, "type must be 'cooperative' or 'dispute'")
	}
}

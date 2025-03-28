package controllers

import (
	"encoding/json"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/ping"
	"net/http"
)

// @Summary Ping the server
// @ID get-ping
// @Produce json
// @Tags Ping
// @Success 200
// @Router /ping [get]
func (h *Handlers) getPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ping.Pong{Message: "pong"}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

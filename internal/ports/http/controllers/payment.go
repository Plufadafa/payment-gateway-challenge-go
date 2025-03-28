package controllers

import (
	"encoding/json"
	"errors"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	"github.com/go-chi/chi/v5"
	"net/http"
)

// @Summary Get a Payment
// @Description Gets a Payment by ID.
// @ID get-payment
// @Tags Payments
// @Accept json
// @Param id path string true "id of the payment"
// @Produce json
// @Success 200 {object} models.Payment
// @Failure 500
// @Failure 404
// @Failure 422
// @Router /api/payments/{id} [GET]
func (h *Handlers) getPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := chi.URLParam(r, "id")
	err := validation.ValidatePaymentID(id)
	if err != nil {
		h.logger.Infof("paymentID: [%s] failed validation: %v", id, err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(err.Error()))
		return
	}

	payment, err := h.paymentsService.GetPayment(id)
	if err != nil {
		h.logger.WithError(err).Infof("error getting payment for paymentID: [%s]", id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if payment == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payment); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// @Summary Process Payment
// @Description Requests a payment be sent to a simulated bank API. Persists the outcome of the payment and  returns the processed payment
// @ID request-payment
// @Tags Payments
// @Accept json
// @Param data body models.ProcessPaymentRequest true "Payment Process Request"
// @Produce json
// @Success 200 {object} models.Payment
// @Failure 500
// @Failure 400
// @Failure 422
// @Router /api/payments [POST]
func (h *Handlers) processPayment(w http.ResponseWriter, r *http.Request) {
	var req models.ProcessPaymentRequest
	paymentID := h.paymentIDCreator.CreatePaymentID()
	h.logger.Infof("received request to process payment for paymentID:[%s]", paymentID)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("error decoding request: %v for paymentID:[%s]", err, paymentID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := h.paymentsValidator.ValidateProcessPaymentRequest(paymentID, &req)
	if err != nil {
		if errors.Is(err, validation.ErrInternal) {
			h.logger.Infof("request to process payment for paymentID: [%s] failed validation with internal error: %v", paymentID, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		h.logger.Infof("request to process payment for paymentID: [%s] failed validation: %v", paymentID, err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(err.Error()))
		return
	}

	processedPayment, err := h.paymentsService.ProcessPayment(paymentID, &req)
	if err != nil {
		h.logger.WithError(err).Errorf("error processing payment with paymentID: [%s]", paymentID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(processedPayment)
}

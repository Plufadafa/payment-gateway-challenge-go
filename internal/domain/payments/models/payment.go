package models

import "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/status"

type (
	Payment struct {
		Id                 string       `json:"id"`
		PaymentStatus      status.State `json:"payment_status"`
		CardNumberLastFour int          `json:"card_number_last_four"`
		ExpiryMonth        int          `json:"expiry_month"`
		ExpiryYear         int          `json:"expiry_year"`
		Currency           string       `json:"currency"`
		Amount             int          `json:"amount"`
	}
)

package models

type (
	//
	ProcessPaymentRequest struct {
		CardNumber  string `json:"card_number"`
		ExpiryMonth int    `json:"expiry_month"`
		ExpiryYear  int    `json:"expiry_year"`
		Currency    string `json:"currency"`
		Amount      int    `json:"amount"`
		CVV         string `json:"cvv"`
	}
)

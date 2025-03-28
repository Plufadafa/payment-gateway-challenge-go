package models

type (
	BankSimPaymentRequest struct {
		CardNumber string `json:"card_number"`
		ExpiryDate string `json:"expiry_date"`
		Currency   string `json:"currency"`
		Amount     int    `json:"amount"`
		CVV        string `json:"cvv"`
	}

	BankSimPaymentResponse struct {
		Authorized        bool   `json:"authorized"`
		AuthorizationCode string `json:"authorization_code"`
	}
)

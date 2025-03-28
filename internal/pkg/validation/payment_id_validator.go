package validation

import (
	"errors"
	"github.com/google/uuid"
)

func ValidatePaymentID(paymentID string) error {
	if paymentID == "" {
		return errors.New("paymentID cannot be empty")
	}

	if err := uuid.Validate(paymentID); err != nil {
		return errors.New("invalid payment ID")
	}
	return nil
}

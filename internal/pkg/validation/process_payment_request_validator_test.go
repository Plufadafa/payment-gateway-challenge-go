package validation_test

import (
	"context"
	"errors"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type (
	ProcessPaymentRequestValidatorSuite struct {
		suite.Suite
		validator validation.IPaymentValidator
	}
)

func TestProcessPaymentRequestValidatorSuite(t *testing.T) {
	suite.Run(t, new(ProcessPaymentRequestValidatorSuite))
}

func (s *ProcessPaymentRequestValidatorSuite) SetupTest() {
	logger := logrus.New().WithContext(context.Background())
	s.validator = validation.NewPaymentValidator(logger)
}

func (s *ProcessPaymentRequestValidatorSuite) TestValidateProcessPaymentRequest() {
	tests := []struct {
		name          string
		request       *models.ProcessPaymentRequest
		expectedError error
	}{
		{
			name: "invalid card number",
			request: &models.ProcessPaymentRequest{
				CardNumber: "123",
			},
			expectedError: errors.New("invalid card number"),
		},
		{
			name: "invalid expiry month",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 123,
			},
			expectedError: errors.New("expiryMonth must be between 1 and 12"),
		},
		{
			name: "invalid expiry Year (before now)",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 11,
				ExpiryYear:  123,
			},
			expectedError: validation.ErrCardExpired,
		},
		{
			name: "invalid expiry Month (current year but month expired)",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: int(time.Now().Month()),
				ExpiryYear:  time.Now().Year(),
			},
			expectedError: validation.ErrCardExpired,
		},
		{
			name: "invalid currency code",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 1,
				ExpiryYear:  2999,
				Currency:    "123",
			},
			expectedError: errors.New("currency not supported"),
		},
		{
			name: "invalid amount",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 1,
				ExpiryYear:  2999,
				Currency:    "EUR",
				Amount:      -1,
			},
			expectedError: errors.New("amount must be greater than zero"),
		},
		{
			name: "invalid CVV (too long)",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 1,
				ExpiryYear:  2999,
				Currency:    "EUR",
				Amount:      1,
				CVV:         "12345",
			},
			expectedError: errors.New("invalid CVV"),
		},
		{
			name: "invalid CVV (too long)",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 1,
				ExpiryYear:  2999,
				Currency:    "EUR",
				Amount:      1,
				CVV:         "12345",
			},
			expectedError: errors.New("invalid CVV"),
		},
		{
			name: "valid request",
			request: &models.ProcessPaymentRequest{
				CardNumber:  "123456789012345",
				ExpiryMonth: 1,
				ExpiryYear:  2999,
				Currency:    "EUR",
				Amount:      1,
				CVV:         "1234",
			},
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		res := s.validator.ValidateProcessPaymentRequest(uuid.New().String(), tt.request)
		if tt.expectedError != nil {
			s.Equal(tt.expectedError, res)
		} else {
			s.Assert().NoError(res)
			s.Assert().Nil(res)
		}
	}
}

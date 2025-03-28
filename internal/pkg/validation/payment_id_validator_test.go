package validation_test

import (
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	PaymentIDValidationSuite struct {
		suite.Suite
	}
)

func TestPaymentValidationSuite(t *testing.T) {
	suite.Run(t, new(PaymentIDValidationSuite))
}

func (s *PaymentIDValidationSuite) TestValidatePaymentID_PaymentIDEmpty_ReturnsError() {
	res := validation.ValidatePaymentID("")
	s.Assert().Error(res)
	s.Assert().Equal("paymentID cannot be empty", res.Error())
}

func (s *PaymentIDValidationSuite) TestValidatePaymentID_InvalidUUID_ReturnsError() {
	res := validation.ValidatePaymentID("12345")
	s.Assert().Error(res)
	s.Assert().Equal("invalid payment ID", res.Error())
}

func (s *PaymentIDValidationSuite) TestValidatePaymentID_ValidUUID_ReturnsNil() {
	res := validation.ValidatePaymentID(uuid.New().String())
	s.Assert().NoError(res)
	s.Assert().Nil(res)

}

package helpers_test

import (
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	PaymentIDCreatorSuite struct {
		suite.Suite
		paymentIDCreator helpers.IPaymentIDCreator
	}
)

func TestPaymentIDCreatorSuite(t *testing.T) {
	suite.Run(t, new(PaymentIDCreatorSuite))
}

func (s *PaymentIDCreatorSuite) SetupTest() {
	s.paymentIDCreator = helpers.NewPaymentIDCreator()
}

func (s *PaymentIDCreatorSuite) TestCreatePaymentID_ReturnsNewUUID() {
	paymentID := s.paymentIDCreator.CreatePaymentID()

	s.Assert().NotPanics(func() {
		uuid.MustParse(paymentID)
	})

}

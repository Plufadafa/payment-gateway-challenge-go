package repository_test

import (
	"context"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/repository"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	RepositorySuite struct {
		suite.Suite
		repo repository.IPaymentsRepository
	}
)

const (
	existingUUID = "04bc7dad-8d7c-437f-b0a1-b72d18b2cde4"
	unusedUUID   = "fe166bbf-fcf3-4a5d-9060-b363269b0aa5"
)

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

func (s *RepositorySuite) SetupTest() {
	logger := logrus.New().WithContext(context.Background())
	s.repo = repository.NewPaymentsRepository(logger)
	s.repo.AddPayment(repository.Payment{
		Id:         existingUUID,
		Authorized: "Authorized",
	})
}

func (s *RepositorySuite) TestAddPayment_PaymentIDExists_ReturnsErrPaymentIDCollision() {
	payment, err := s.repo.AddPayment(repository.Payment{
		Id: existingUUID,
	})

	s.Error(err)
	s.Nil(payment)
	s.Assert().ErrorIs(err, repository.ErrPaymentIDCollision)
}

func (s *RepositorySuite) TestAddPayment_PaymentIDDoesNotExist_AddsPaymentAndReturnsPersistedModel() {
	payment, err := s.repo.AddPayment(repository.Payment{
		Id: unusedUUID,
	})

	s.Assert().NoError(err)
	s.Assert().Equal(unusedUUID, payment.Id)
}

func (s *RepositorySuite) TestGetPayment_PaymentDesNotExist_ReturnsNil() {
	payment := s.repo.GetPayment(unusedUUID)

	s.Assert().Nil(payment)
}

func (s *RepositorySuite) TestGetPayment_PaymentExists_ReturnsNil() {
	payment := s.repo.GetPayment(existingUUID)

	s.Assert().NotNil(payment)
	s.Assert().Equal(payment.Authorized, "Authorized")
}

package payments

import (
	"errors"
	"fmt"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/helpers"
	sharedmodels "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/status"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients"
	banksimapimodels "github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/repository"
	"github.com/sirupsen/logrus"
	"strconv"
)

//go:generate mockgen -destination=./mocks/IService.go -package mocks . IService
type (
	Service struct {
		paymentsRepository repository.IPaymentsRepository
		bankSimApiClient   clients.IBankSimAPI
		paymentsValidator  validation.IPaymentValidator
		paymentIDCreator   helpers.IPaymentIDCreator
		logger             *logrus.Entry
	}

	IService interface {
		GetPayment(id string) (*models.Payment, error)
		ProcessPayment(paymentID string, processPaymentRequest *sharedmodels.ProcessPaymentRequest) (*models.Payment, error)
	}
)

const (
	maxPersistenceAttempt = 5
)

func NewService(paymentsRepository repository.IPaymentsRepository, bankSimApiClient clients.IBankSimAPI, paymentsValidator validation.IPaymentValidator, paymentIDCreator helpers.IPaymentIDCreator, logger *logrus.Entry) IService {
	return &Service{
		paymentsRepository: paymentsRepository,
		bankSimApiClient:   bankSimApiClient,
		paymentsValidator:  paymentsValidator,
		paymentIDCreator:   paymentIDCreator,
		logger:             logger,
	}
}

func (s *Service) GetPayment(paymentID string) (*models.Payment, error) {
	err := validation.ValidatePaymentID(paymentID)
	if err != nil {
		s.logger.WithError(err).Errorf("paymentID: [%s] failed validation", paymentID)
		return nil, err
	}

	repoPayment := s.paymentsRepository.GetPayment(paymentID)
	if repoPayment == nil {
		return nil, nil
	}

	paymentStatus, err := status.NewState(repoPayment.Authorized)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to translate persisted status: [%s] for paymentID: [%s]", repoPayment.Authorized, paymentID)
		return nil, err
	}

	return &models.Payment{
		Id:                 repoPayment.Id,
		PaymentStatus:      paymentStatus,
		CardNumberLastFour: repoPayment.CardNumberLastFour,
		ExpiryMonth:        repoPayment.ExpiryMonth,
		ExpiryYear:         repoPayment.ExpiryYear,
		Currency:           repoPayment.Currency,
		Amount:             repoPayment.Amount,
	}, nil
}

func (s *Service) ProcessPayment(paymentID string, processPaymentRequest *sharedmodels.ProcessPaymentRequest) (*models.Payment, error) {
	err := s.paymentsValidator.ValidateProcessPaymentRequest(paymentID, processPaymentRequest)
	if err != nil {
		s.logger.WithError(err).Error("failed to validate process payment request")
		return nil, err
	}

	if err = validation.ValidatePaymentID(paymentID); err != nil {
		s.logger.WithError(err).Errorf("paymentID:[%s] failed validation", paymentID)
		return nil, err
	}

	bankSimRequest := banksimapimodels.BankSimPaymentRequest{
		CardNumber: processPaymentRequest.CardNumber,
		Currency:   processPaymentRequest.Currency,
		Amount:     processPaymentRequest.Amount,
		CVV:        processPaymentRequest.CVV,
		ExpiryDate: s.formatExpiryDate(processPaymentRequest.ExpiryMonth, processPaymentRequest.ExpiryYear),
	}

	bankSimResponse, err := s.bankSimApiClient.ForwardPaymentRequest(paymentID, &bankSimRequest)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to forward payment request for paymentID: [%s]", paymentID)
		return nil, err
	}

	lastFourCardNum := processPaymentRequest.CardNumber[len(processPaymentRequest.CardNumber)-4:]

	bankSimResponseStatus := status.StateFromIsAuthorized(bankSimResponse.Authorized)

	paymentToPersist := repository.Payment{
		Id:                 paymentID,
		Authorized:         bankSimResponseStatus.String(),
		CardNumberLastFour: lastFourCardNum,
		ExpiryMonth:        processPaymentRequest.ExpiryMonth,
		ExpiryYear:         processPaymentRequest.ExpiryYear,
		Currency:           processPaymentRequest.Currency,
		Amount:             processPaymentRequest.Amount,
	}

	persistedPayment := s.persistPayment(paymentID, paymentToPersist)
	persistedPaymentStatus, err := status.NewState(persistedPayment.Authorized)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to translate persisted status: [%s] for persisted paymentID: [%s]", persistedPayment.Authorized, persistedPayment.Id)
		return nil, err
	}

	return &models.Payment{
		Id:                 persistedPayment.Id,
		PaymentStatus:      persistedPaymentStatus,
		CardNumberLastFour: persistedPayment.CardNumberLastFour,
		ExpiryMonth:        persistedPayment.ExpiryMonth,
		ExpiryYear:         persistedPayment.ExpiryYear,
		Currency:           persistedPayment.Currency,
		Amount:             persistedPayment.Amount,
	}, nil
}

func (s *Service) formatExpiryDate(month, year int) string {
	formattedMonth := strconv.Itoa(month)

	if month <= 9 {
		formattedMonth = "0" + formattedMonth
	}

	return fmt.Sprintf("%s/%v", formattedMonth, year)
}

func (s *Service) persistPayment(paymentID string, paymentToPersist repository.Payment) *repository.Payment {
	persistedPayment, err := s.paymentsRepository.AddPayment(paymentToPersist)
	if err != nil {
		if errors.Is(repository.ErrPaymentIDCollision, err) {
			s.logger.Infof("uuid collision occurred for paymentID: [%s], reattempting repository write", paymentID)
			newPaymentID := s.paymentIDCreator.CreatePaymentID()
			s.logger.Infof("reattempting payment persistence. Reassigning paymentID: [%s] to [%s]", paymentToPersist.Id, newPaymentID)
			paymentToPersist.Id = newPaymentID
		} else {
			s.logger.Errorf("error occurred during AddPayment for paymentID [%s]", paymentID)
		}
		return s.retryPersistPayment(paymentID, &paymentToPersist)
	}
	return persistedPayment
}

func (s *Service) retryPersistPayment(originalPaymentID string, paymentToPersist *repository.Payment) *repository.Payment {
	for i := 0; i < maxPersistenceAttempt; i++ {
		persistedPayment, err := s.paymentsRepository.AddPayment(*paymentToPersist)
		if err != nil {
			if errors.Is(repository.ErrPaymentIDCollision, err) {
				s.logger.Errorf("uuid collision occurred for paymentID: [%s] on retry attempt: [%s]", paymentToPersist.Id, strconv.Itoa(i))
				newPaymentID := s.paymentIDCreator.CreatePaymentID()
				s.logger.Infof("reattempting payment persistence. Reassigning paymentID: [%s] to [%s]", paymentToPersist.Id, newPaymentID)
				paymentToPersist.Id = newPaymentID
			} else {
				s.logger.Errorf("error occurred during AddPayment for paymentID [%s]", paymentToPersist.Id)
			}
			continue
		}

		s.logger.Infof("payment persistence for paymentID: [%s] succeeded on attempt: [%s]", paymentToPersist.Id, strconv.Itoa(i))
		return persistedPayment
	}

	s.logger.Errorf("failed to persist payment with original paymentID: [%s] maximum number of times: [%v] triggering page and returning payment with original paymentID", originalPaymentID, maxPersistenceAttempt)
	// trigger page to alert team to failed persistence of payment with X fields provided within PCI regulation to identify payment. Will return the payment as if it had been persisted
	// to our database so Merchant can see success.
	return paymentToPersist
}

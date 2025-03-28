package payments_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments/models"
	helpermocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/helpers/mocks"
	pkgmodels "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/status"
	validationmocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation/mocks"
	clientmocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/mocks"
	banksimmodels "github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/models"
	repo "github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/repository"
	repomocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"strconv"
	"testing"
	"time"
)

type (
	ServiceSuite struct {
		suite.Suite
		mockPaymentsRepository *repomocks.MockIPaymentsRepository
		mockBankSimApiClient   *clientmocks.MockIBankSimAPI
		mockPaymentsValidator  *validationmocks.MockIPaymentValidator
		mockPaymentIDCreator   *helpermocks.MockIPaymentIDCreator
		service                payments.IService
		formattedExpiryDate    string
		validProcessRequest    pkgmodels.ProcessPaymentRequest
	}
)

func (s *ServiceSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.mockPaymentsRepository = repomocks.NewMockIPaymentsRepository(ctrl)
	s.mockBankSimApiClient = clientmocks.NewMockIBankSimAPI(ctrl)
	s.mockPaymentsValidator = validationmocks.NewMockIPaymentValidator(ctrl)
	s.mockPaymentIDCreator = helpermocks.NewMockIPaymentIDCreator(ctrl)
	logger := logrus.New().WithContext(context.Background())
	s.formattedExpiryDate = s.formatExpiryDate(int(time.Now().Month()), time.Now().Year())
	s.validProcessRequest = pkgmodels.ProcessPaymentRequest{
		CardNumber:  "12345678912345",
		ExpiryMonth: int(time.Now().Month()),
		ExpiryYear:  time.Now().Year(),
		Currency:    "EUR",
		Amount:      10,
		CVV:         "465",
	}

	s.service = payments.NewService(s.mockPaymentsRepository, s.mockBankSimApiClient, s.mockPaymentsValidator, s.mockPaymentIDCreator, logger)
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) TestGetPayment_PaymentIDFound_ReturnsPayment() {
	expectedPayment := models.Payment{
		Id:                 uuid.New().String(),
		PaymentStatus:      status.StateAuthorized,
		CardNumberLastFour: "1234",
		ExpiryMonth:        10,
		ExpiryYear:         2099,
		Currency:           "EUR",
		Amount:             10,
	}

	repoPayment := repo.Payment{
		Id:                 expectedPayment.Id,
		Authorized:         expectedPayment.PaymentStatus.String(),
		CardNumberLastFour: expectedPayment.CardNumberLastFour,
		ExpiryMonth:        expectedPayment.ExpiryMonth,
		ExpiryYear:         expectedPayment.ExpiryYear,
		Currency:           expectedPayment.Currency,
		Amount:             expectedPayment.Amount,
	}

	s.mockPaymentsRepository.EXPECT().GetPayment(expectedPayment.Id).Return(&repoPayment)

	res, err := s.service.GetPayment(expectedPayment.Id)

	s.NoError(err)
	s.EqualValues(&expectedPayment, res)
}

func (s *ServiceSuite) TestGetPayment_InvalidPaymentID_ReturnsError() {
	s.mockPaymentsRepository.EXPECT().GetPayment(gomock.Any()).Times(0)

	res, err := s.service.GetPayment("asdf")

	s.Assert().Error(err)
	s.Assert().Empty(res)
}

func (s *ServiceSuite) TestGetPayment_RepoReturnsNil_ReturnsNil() {
	paymentID := uuid.New().String()
	s.mockPaymentsRepository.EXPECT().GetPayment(paymentID).Return(nil)

	res, err := s.service.GetPayment(paymentID)

	s.NoError(err)
	s.Nil(res)
}

func (s *ServiceSuite) TestProcessPayment_RequestInvalid_ReturnsError() {
	processRequest := pkgmodels.ProcessPaymentRequest{
		CardNumber: "1234567",
	}
	paymentID := uuid.New().String()

	s.mockPaymentsValidator.EXPECT().ValidateProcessPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&processRequest)).Return(errors.New("error"))
	res, err := s.service.ProcessPayment(paymentID, &processRequest)

	s.Assert().Error(err)
	s.Assert().Equal("error", err.Error())
	s.Assert().Empty(res)
}

func (s *ServiceSuite) TestProcessPayment_PaymentIDInvalid_ReturnsError() {
	paymentID := "123"
	s.mockPaymentsValidator.EXPECT().ValidateProcessPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&s.validProcessRequest)).Return(nil)
	res, err := s.service.ProcessPayment(paymentID, &s.validProcessRequest)

	s.Assert().Error(err)
	s.Assert().Empty(res)
}

func (s *ServiceSuite) TestProcessPayment_BankSimClientErrors_ReturnsError() {
	bankSimRequest := banksimmodels.BankSimPaymentRequest{
		CardNumber: s.validProcessRequest.CardNumber,
		ExpiryDate: s.formattedExpiryDate,
		Currency:   s.validProcessRequest.Currency,
		Amount:     s.validProcessRequest.Amount,
		CVV:        s.validProcessRequest.CVV,
	}
	paymentID := uuid.New().String()
	s.mockPaymentsValidator.EXPECT().ValidateProcessPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&s.validProcessRequest)).Return(nil)
	s.mockBankSimApiClient.EXPECT().ForwardPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&bankSimRequest)).Return(nil, errors.New("error"))
	res, err := s.service.ProcessPayment(paymentID, &s.validProcessRequest)

	s.Assert().Error(err)
	s.Assert().Empty(res)
}

func (s *ServiceSuite) TestProcessPayment_PaymentProcessedAndPersisted_ReturnsCompletedPayment() {
	bankSimRequest := banksimmodels.BankSimPaymentRequest{
		CardNumber: s.validProcessRequest.CardNumber,
		ExpiryDate: s.formattedExpiryDate,
		Currency:   s.validProcessRequest.Currency,
		Amount:     s.validProcessRequest.Amount,
		CVV:        s.validProcessRequest.CVV,
	}

	bankSimResponse := banksimmodels.BankSimPaymentResponse{
		Authorized:        true,
		AuthorizationCode: uuid.New().String(),
	}
	paymentID := uuid.New().String()

	expectedResponse := models.Payment{
		Id:                 paymentID,
		PaymentStatus:      status.StateAuthorized,
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	repoPersistencePayment := repo.Payment{
		Id:                 paymentID,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	s.mockPaymentsValidator.EXPECT().ValidateProcessPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&s.validProcessRequest)).Return(nil)
	s.mockBankSimApiClient.EXPECT().ForwardPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&bankSimRequest)).Return(&bankSimResponse, nil)
	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(repoPersistencePayment)).Return(&repoPersistencePayment, nil)
	res, err := s.service.ProcessPayment(paymentID, &s.validProcessRequest)

	s.Assert().NoError(err)
	s.Assert().Nil(err)
	s.Assert().EqualValues(res, &expectedResponse)
}

func (s *ServiceSuite) TestProcessPayment_UUIDCollision_RetriesMaxFiveTimes() {
	bankSimRequest := banksimmodels.BankSimPaymentRequest{
		CardNumber: s.validProcessRequest.CardNumber,
		ExpiryDate: s.formattedExpiryDate,
		Currency:   s.validProcessRequest.Currency,
		Amount:     s.validProcessRequest.Amount,
		CVV:        s.validProcessRequest.CVV,
	}

	bankSimResponse := banksimmodels.BankSimPaymentResponse{
		Authorized:        true,
		AuthorizationCode: uuid.New().String(),
	}
	paymentID := uuid.New().String()

	repoPersistencePayment := repo.Payment{
		Id:                 paymentID,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	remadePaymentIDOne := uuid.New().String()
	remadePaymentIDTwo := uuid.New().String()
	remadePaymentIDThree := uuid.New().String()
	remadePaymentIDFour := uuid.New().String()
	remadePaymentIDFive := uuid.New().String()

	repoPersistencePaymentRetryOne := repo.Payment{
		Id:                 remadePaymentIDOne,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	repoPersistencePaymentRetryTwo := repo.Payment{
		Id:                 remadePaymentIDTwo,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	repoPersistencePaymentRetryThree := repo.Payment{
		Id:                 remadePaymentIDThree,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	repoPersistencePaymentRetryFour := repo.Payment{
		Id:                 remadePaymentIDFour,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	successRepoPersistencePaymentRetryFive := repo.Payment{
		Id:                 remadePaymentIDFive,
		Authorized:         status.StateAuthorized.String(),
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	expectedResponse := models.Payment{
		Id:                 remadePaymentIDFive,
		PaymentStatus:      status.StateAuthorized,
		CardNumberLastFour: "2345",
		ExpiryMonth:        s.validProcessRequest.ExpiryMonth,
		ExpiryYear:         s.validProcessRequest.ExpiryYear,
		Currency:           s.validProcessRequest.Currency,
		Amount:             s.validProcessRequest.Amount,
	}

	s.mockPaymentIDCreator.EXPECT().CreatePaymentID().Return(remadePaymentIDOne).Times(1)
	s.mockPaymentIDCreator.EXPECT().CreatePaymentID().Return(remadePaymentIDTwo).Times(1)
	s.mockPaymentIDCreator.EXPECT().CreatePaymentID().Return(remadePaymentIDThree).Times(1)
	s.mockPaymentIDCreator.EXPECT().CreatePaymentID().Return(remadePaymentIDFour).Times(1)
	s.mockPaymentIDCreator.EXPECT().CreatePaymentID().Return(remadePaymentIDFive).Times(1)

	s.mockPaymentsValidator.EXPECT().ValidateProcessPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&s.validProcessRequest)).Return(nil)
	s.mockBankSimApiClient.EXPECT().ForwardPaymentRequest(gomock.Eq(paymentID), gomock.Eq(&bankSimRequest)).Return(&bankSimResponse, nil)
	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(repoPersistencePayment)).Return(nil, repo.ErrPaymentIDCollision).Times(1)

	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(repoPersistencePaymentRetryOne)).Return(nil, repo.ErrPaymentIDCollision).Times(1)
	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(repoPersistencePaymentRetryTwo)).Return(nil, repo.ErrPaymentIDCollision).Times(1)
	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(repoPersistencePaymentRetryThree)).Return(nil, repo.ErrPaymentIDCollision).Times(1)
	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(repoPersistencePaymentRetryFour)).Return(nil, repo.ErrPaymentIDCollision).Times(1)
	s.mockPaymentsRepository.EXPECT().AddPayment(gomock.Eq(successRepoPersistencePaymentRetryFive)).Return(&successRepoPersistencePaymentRetryFive, nil).Times(1)

	res, err := s.service.ProcessPayment(paymentID, &s.validProcessRequest)

	s.Assert().NoError(err)
	s.Assert().Nil(err)
	s.Assert().EqualValues(res, &expectedResponse)
}

func (s *ServiceSuite) formatExpiryDate(month, year int) string {
	formattedMonth := strconv.Itoa(month)

	if month < 9 {
		formattedMonth = "0" + formattedMonth
	}

	return fmt.Sprintf("%s/%v", formattedMonth, year)
}

package controllers_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	paymentmocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments/mocks"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments/models"
	helpermocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/helpers/mocks"
	pkgmodels "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/status"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	validationmocks "github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation/mocks"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/controllers"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"testing"
)

type (
	HttpSuite struct {
		suite.Suite
		handlers              controllers.IHandlers
		paymentServiceMock    *paymentmocks.MockIService
		paymentsValidatorMock *validationmocks.MockIPaymentValidator
		paymentIDCreator      *helpermocks.MockIPaymentIDCreator
		app                   *httptest.Server
		chiRouter             *chi.Mux
		logger                *logrus.Entry
	}
)

func TestHttpSuite(t *testing.T) {
	suite.Run(t, new(HttpSuite))
}

func (s *HttpSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	ctx := context.Background()
	s.logger = logrus.New().WithContext(ctx)
	s.chiRouter = chi.NewRouter()
	s.paymentServiceMock = paymentmocks.NewMockIService(ctrl)
	s.paymentsValidatorMock = validationmocks.NewMockIPaymentValidator(ctrl)
	s.paymentIDCreator = helpermocks.NewMockIPaymentIDCreator(ctrl)
	s.handlers = controllers.NewHandlers(s.paymentServiceMock, s.paymentsValidatorMock, s.paymentIDCreator, s.logger)
	s.handlers.SetupRoutes(s.chiRouter)
	s.app = httptest.NewServer(s.chiRouter)
}

func (s *HttpSuite) TestGetPing_Returns200WithMessage() {
	req := httptest.NewRequest("GET", s.app.URL+"/ping", nil)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	var responseMessage struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseMessage); err != nil {
		s.T().Fatal(err)
	}

	s.Assert().Equal(http.StatusOK, response.StatusCode)
	s.Assert().Equal(responseMessage.Message, "pong")
}

func (s *HttpSuite) TestGetPayment_PaymentExists_Returns200WithPayment() {
	paymentID := uuid.New().String()
	expectedPayment := &models.Payment{
		Id:                 paymentID,
		PaymentStatus:      status.StateAuthorized,
		CardNumberLastFour: "1234",
		ExpiryMonth:        01,
		ExpiryYear:         2028,
		Currency:           "EUR",
		Amount:             100,
	}

	s.paymentServiceMock.EXPECT().GetPayment(paymentID).Return(expectedPayment, nil)

	req := httptest.NewRequest("GET", s.app.URL+"/api/payments/"+paymentID, nil)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	var responsePayment models.Payment

	if err := json.NewDecoder(response.Body).Decode(&responsePayment); err != nil {
		s.T().Fatal(err)
	}

	s.Assert().Equal(http.StatusOK, response.StatusCode)
	s.Assert().EqualValues(*expectedPayment, responsePayment)
}

func (s *HttpSuite) TestGetPayment_PaymentDoesNotExist_Returns404() {
	paymentID := uuid.New().String()

	s.paymentServiceMock.EXPECT().GetPayment(paymentID).Return(nil, nil)

	req := httptest.NewRequest("GET", s.app.URL+"/api/payments/"+paymentID, nil)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	s.Assert().Equal(http.StatusNotFound, response.StatusCode)
}

func (s *HttpSuite) TestGetPayment_PaymentServiceErrors_Returns500() {
	paymentID := uuid.New().String()

	s.paymentServiceMock.EXPECT().GetPayment(paymentID).Return(nil, errors.New("service error"))

	req := httptest.NewRequest("GET", s.app.URL+"/api/payments/"+paymentID, nil)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	s.Assert().Equal(http.StatusInternalServerError, response.StatusCode)
}

func (s *HttpSuite) TestGetPayment_PaymentIDFailsValidation_Returns422() {
	paymentID := "checkout.com"

	s.paymentServiceMock.EXPECT().GetPayment(paymentID).Times(0)

	req := httptest.NewRequest("GET", s.app.URL+"/api/payments/"+paymentID, nil)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body) //nolint errcheck
	responseString := buf.String()

	s.Assert().Equal(http.StatusUnprocessableEntity, response.StatusCode)
	s.Assert().EqualValues("invalid payment ID", responseString)
}

func (s *HttpSuite) TestProcessPayment_RequestValid_Returns200WithPayment() {
	expectedPayment := &models.Payment{
		PaymentStatus:      status.StateAuthorized,
		CardNumberLastFour: "1234",
		ExpiryMonth:        01,
		ExpiryYear:         2028,
		Currency:           "EUR",
		Amount:             100,
	}

	processPaymentRequest := pkgmodels.ProcessPaymentRequest{
		CardNumber:  "12345678912345",
		ExpiryMonth: 01,
		ExpiryYear:  2028,
		Currency:    "EUR",
		Amount:      100,
		CVV:         "123",
	}

	s.paymentIDCreator.EXPECT().CreatePaymentID().Return(uuid.New().String())
	s.paymentsValidatorMock.EXPECT().ValidateProcessPaymentRequest(gomock.Any(), gomock.Eq(&processPaymentRequest)).Return(nil)
	s.paymentServiceMock.EXPECT().ProcessPayment(gomock.Any(), gomock.Eq(&processPaymentRequest)).Return(expectedPayment, nil)

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	err := json.NewEncoder(bw).Encode(processPaymentRequest)
	if err != nil {
		s.T().Fatal(err)
	}
	err = bw.Flush()
	if err != nil {
		s.T().Fatal(err)
	}
	byteReader := bytes.NewReader(b.Bytes())

	req := httptest.NewRequest("POST", s.app.URL+"/api/payments", byteReader)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	var responsePayment models.Payment

	if err := json.NewDecoder(response.Body).Decode(&responsePayment); err != nil {
		s.T().Fatal(err)
	}

	s.Assert().Equal(http.StatusOK, response.StatusCode)
	s.Assert().EqualValues(*expectedPayment, responsePayment)
}

func (s *HttpSuite) TestProcessPayment_RequestFailsValidation_Returns422() {
	processPaymentRequest := pkgmodels.ProcessPaymentRequest{
		CardNumber:  "123452345",
		ExpiryMonth: 11,
		ExpiryYear:  202,
		Currency:    "ABC",
		Amount:      -100,
		CVV:         "111123",
	}

	s.paymentIDCreator.EXPECT().CreatePaymentID().Return(uuid.New().String())
	s.paymentsValidatorMock.EXPECT().ValidateProcessPaymentRequest(gomock.Any(), gomock.Eq(&processPaymentRequest)).Return(errors.New("failed validation"))
	s.paymentServiceMock.EXPECT().ProcessPayment(gomock.Any(), gomock.Any()).Times(0)

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	err := json.NewEncoder(bw).Encode(processPaymentRequest)
	if err != nil {
		s.T().Fatal(err)
	}
	err = bw.Flush()
	if err != nil {
		s.T().Fatal(err)
	}
	byteReader := bytes.NewReader(b.Bytes())

	req := httptest.NewRequest("POST", s.app.URL+"/api/payments", byteReader)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	s.Assert().Equal(http.StatusUnprocessableEntity, response.StatusCode)
}

func (s *HttpSuite) TestProcessPayment_ValidatorReturnsErrInternal_Returns500() {
	processPaymentRequest := pkgmodels.ProcessPaymentRequest{
		CardNumber:  "123452345",
		ExpiryMonth: 11,
		ExpiryYear:  202,
		Currency:    "ABC",
		Amount:      -100,
		CVV:         "111123",
	}

	s.paymentIDCreator.EXPECT().CreatePaymentID().Return(uuid.New().String())
	s.paymentsValidatorMock.EXPECT().ValidateProcessPaymentRequest(gomock.Any(), gomock.Eq(&processPaymentRequest)).Return(validation.ErrInternal)
	s.paymentServiceMock.EXPECT().ProcessPayment(gomock.Any(), gomock.Any()).Times(0)

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	err := json.NewEncoder(bw).Encode(processPaymentRequest)
	if err != nil {
		s.T().Fatal(err)
	}
	err = bw.Flush()
	if err != nil {
		s.T().Fatal(err)
	}
	byteReader := bytes.NewReader(b.Bytes())

	req := httptest.NewRequest("POST", s.app.URL+"/api/payments", byteReader)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	s.Assert().Equal(http.StatusInternalServerError, response.StatusCode)
}

func (s *HttpSuite) TestProcessPayment_PaymentServiceReturnsError_Returns500() {
	processPaymentRequest := pkgmodels.ProcessPaymentRequest{
		CardNumber:  "12345678912345",
		ExpiryMonth: 1,
		ExpiryYear:  2028,
		Currency:    "EUR",
		Amount:      100,
		CVV:         "123",
	}

	pid := uuid.New().String()
	s.paymentIDCreator.EXPECT().CreatePaymentID().Return(pid)
	s.paymentsValidatorMock.EXPECT().ValidateProcessPaymentRequest(gomock.Eq(pid), gomock.Eq(&processPaymentRequest)).Return(nil)
	s.paymentServiceMock.EXPECT().ProcessPayment(gomock.Any(), gomock.Eq(&processPaymentRequest)).Return(nil, errors.New("payment service error"))

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	err := json.NewEncoder(bw).Encode(processPaymentRequest)
	if err != nil {
		s.T().Fatal(err)
	}
	err = bw.Flush()
	if err != nil {
		s.T().Fatal(err)
	}
	byteReader := bytes.NewReader(b.Bytes())

	req := httptest.NewRequest("POST", s.app.URL+"/api/payments", byteReader)
	responseRecorder := httptest.NewRecorder()
	s.chiRouter.ServeHTTP(responseRecorder, req)
	response := responseRecorder.Result()

	s.Assert().Equal(http.StatusInternalServerError, response.StatusCode)
}

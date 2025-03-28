package clients

import (
	"bytes"
	"errors"
	"io"

	"encoding/json"
	"net/http"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/config"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/models"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -destination=./mocks/IBankSimAPI.go -package mocks . IBankSimAPI
type (
	BankSimAPI struct {
		httpClient http.Client
		cfg        *config.Config
		logger     *logrus.Entry
	}

	IBankSimAPI interface {
		ForwardPaymentRequest(request *models.BankSimPaymentRequest) (*models.BankSimPaymentResponse, error)
	}

	ErrPaymentRequest struct {
		message string
	}
)

func (e *ErrPaymentRequest) Error() string {
	return e.message
}

func NewBankSimAPI(httpClient http.Client, cfg *config.Config, logger *logrus.Entry) IBankSimAPI {
	return &BankSimAPI{
		httpClient: httpClient,
		cfg:        cfg,
		logger:     logger,
	}
}

func (b *BankSimAPI) ForwardPaymentRequest(request *models.BankSimPaymentRequest) (*models.BankSimPaymentResponse, error) {
	if request == nil {
		return nil, errors.New("bankSimPaymentRequest cannot be nil")
	}

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.New("error marshalling payment request")
	}

	req, err := http.NewRequest(http.MethodPost, b.cfg.BankSimURL+"/payments", bytes.NewReader(bodyBytes))

	if err != nil {
		b.logger.WithError(err).Error("error creating http request for new payment request")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		b.logger.WithError(err).Error("error performing http request for new payment request")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bod, err := io.ReadAll(resp.Body)
		if err != nil {
			b.logger.WithError(err).Error("error reading response body")
			return nil, err
		}
		responseBodyString := bytes.NewBuffer(bod).String()
		return nil, &ErrPaymentRequest{message: responseBodyString}
	}

	var response models.BankSimPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		b.logger.WithError(err).Error("error decoding response body")
	}

	return &response, nil
}

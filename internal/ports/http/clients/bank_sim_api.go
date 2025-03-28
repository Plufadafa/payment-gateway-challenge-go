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
		httpClient             http.Client
		cfg                    *config.Config
		bankSimApiRetryHandler IBankSimApiRetryHandler
		logger                 *logrus.Entry
	}

	IBankSimAPI interface {
		ForwardPaymentRequest(paymentID string, request *models.BankSimPaymentRequest) (*models.BankSimPaymentResponse, error)
	}

	ErrPaymentRequest struct {
		message string
	}
)

func (e *ErrPaymentRequest) Error() string {
	return e.message
}

func NewBankSimAPI(httpClient http.Client, bankSimApiRetryHandler IBankSimApiRetryHandler, cfg *config.Config, logger *logrus.Entry) IBankSimAPI {
	return &BankSimAPI{
		httpClient:             httpClient,
		bankSimApiRetryHandler: bankSimApiRetryHandler,
		cfg:                    cfg,
		logger:                 logger,
	}
}

func (b *BankSimAPI) ForwardPaymentRequest(paymentID string, request *models.BankSimPaymentRequest) (*models.BankSimPaymentResponse, error) {
	if request == nil {
		return nil, errors.New("bankSimPaymentRequest cannot be nil")
	}

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.New("error marshalling payment request")
	}

	requestURL := b.cfg.BankSimURL + "/payments"
	bodyBytesReader := bytes.NewReader(bodyBytes)

	req, err := http.NewRequest(http.MethodPost, requestURL, bodyBytesReader)
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

	var response models.BankSimPaymentResponse

	if resp.StatusCode != http.StatusOK {
		// begins exponential backoff in the event server is temporarily unavailable
		if resp.StatusCode == http.StatusServiceUnavailable {
			r, err := b.bankSimApiRetryHandler.BeginRetry(paymentID, requestURL, bodyBytesReader)
			if err != nil {
				b.logger.WithError(err).Error("error retrying http request for new payment request")
				return nil, err
			}
			if r != nil {
				return r, nil
			}
		}
		// in the event response code is for anything other than a 503, we don't want to retry
		bod, err := io.ReadAll(resp.Body)
		if err != nil {
			b.logger.WithError(err).Error("error reading response body")
			return nil, err
		}
		// include the response from bank api in error
		responseBodyString := bytes.NewBuffer(bod).String()
		return nil, &ErrPaymentRequest{message: responseBodyString}
	}

	// status ok, decode the response to return
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		b.logger.WithError(err).Error("error decoding response body")
	}

	return &response, nil
}

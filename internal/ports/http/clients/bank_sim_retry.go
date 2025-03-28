package clients

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/config"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/models"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"net/http"
	"time"
)

type (
	BankSimApiRetryHandler struct {
		httpClient http.Client
		cfg        *config.Config
		logger     *logrus.Entry
	}

	IBankSimApiRetryHandler interface {
		BeginRetry(paymentID, url string, request *models.BankSimPaymentRequest) (*models.BankSimPaymentResponse, error)
	}
)

const maxRetryLimit = 5

var baseDelay = 1 * time.Second

func NewBankSimApiRetryHandler(httpClient http.Client, cfg *config.Config, logger *logrus.Entry) IBankSimApiRetryHandler {
	return &BankSimApiRetryHandler{
		httpClient: httpClient,
		cfg:        cfg,
		logger:     logger,
	}
}

func (b *BankSimApiRetryHandler) BeginRetry(paymentID, url string, request *models.BankSimPaymentRequest) (*models.BankSimPaymentResponse, error) {
	for i := 0; i < maxRetryLimit; i++ {
		b.logger.Infof("attempting payment request retry for paymentID: [%s] attempt: [%v]", paymentID, i)
		resp, shouldRetry, err := b.performRequest(paymentID, url, i, request)
		// shortcuts to continue the exponential backoff with a wait
		if shouldRetry {
			secRetry := math.Pow(2, float64(i))
			b.logger.Infof("retrying payment request for paymentID: [%s] in [%f] seconds", paymentID, secRetry)
			delay := time.Duration(secRetry) * baseDelay
			time.Sleep(delay)
			continue
		}

		// should not retry, if error occurred do not continue exponential backoff
		if err != nil {
			return nil, err
		}

		// retry was a success, return the completed payment
		return resp, nil
	}

	return nil, errors.New("max retry limit reached")
}

// isolates the request logic from retry loop, ensures defers are completed so no resource leaks. Easier to maintain.
func (b *BankSimApiRetryHandler) performRequest(paymentID, url string, retryAttempt int, request *models.BankSimPaymentRequest) (response *models.BankSimPaymentResponse, attemptRetry bool, err error) {
	var paymentResponse models.BankSimPaymentResponse
	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, false, errors.New("error marshalling payment request")
	}
	bodyBytesReader := bytes.NewReader(bodyBytes)
	req, err := http.NewRequest(http.MethodPost, url, bodyBytesReader)
	if err != nil {
		b.logger.WithError(err).Error("error creating http request for new payment request")
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		b.logger.WithError(err).Error("error performing http request for new payment request")
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// this is the only status code which should trigger a retry
		if resp.StatusCode == http.StatusServiceUnavailable {
			b.logger.Infof("paymentID: [%s] payment process resulted in 503 on retry attempt: [%v]", paymentID, retryAttempt)
			return nil, true, nil
		}

		// don't continue retry loop in the event of any other errors
		bod, err := io.ReadAll(resp.Body)
		if err != nil {
			b.logger.WithError(err).Errorf("error reading response body for paymentID: [%s]", paymentID)
			return nil, false, err
		}
		responseBodyString := bytes.NewBuffer(bod).String()

		b.logger.Infof("paymentID: [%s] payment process resulted in error: [%v] error message: [%s]", paymentID, resp.StatusCode, responseBodyString)
		return nil, false, nil
	}

	// retry succeeded, decode the response and return false for attemptRetry
	if err := json.NewDecoder(resp.Body).Decode(&paymentResponse); err != nil {
		b.logger.WithError(err).Errorf("error decoding response body for paymentID: [%s]", paymentID)
		return nil, false, err
	}
	return &paymentResponse, false, nil
}

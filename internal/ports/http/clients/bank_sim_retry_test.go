package clients_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/config"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/models"
)

func TestBeginRetry(t *testing.T) {
	tests := []struct {
		name                    string
		succeedsAfterAttempts   int
		badRequestafterAttempts int
		expectedError           error
		expectedResponse        *models.BankSimPaymentResponse
	}{
		{
			name:                    "retry fails after 2 attempts",
			badRequestafterAttempts: 2,
			expectedResponse:        nil,
			expectedError:           clients.NewErrPaymentRequest("bad request"),
		},
		{
			name:                  "retry succeeds after 2 attempts",
			succeedsAfterAttempts: 2,
			expectedResponse:      &authorizedResponse,
		},
		{
			name:                    "retry hits max failures",
			badRequestafterAttempts: 3,
			expectedResponse:        nil,
			expectedError:           errors.New("max retry limit reached"),
		},
	}

	for _, tt := range tests {
		retryCounter := 0
		t.Run(tt.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "POST":
					assert.Equal(t, "", r.URL.RawQuery)
					if retryCounter < tt.succeedsAfterAttempts || retryCounter < tt.badRequestafterAttempts {
						w.WriteHeader(http.StatusServiceUnavailable)
						retryCounter++
						return
					}

					if tt.badRequestafterAttempts > 0 {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte("bad request")) //nolint errcheck
						return
					}

					w.WriteHeader(http.StatusOK)
					m, err := json.Marshal(tt.expectedResponse)
					if err != nil {
						t.Errorf("error marshalling response: %v", err)
						return
					}
					w.Write(m) //nolint errcheck
				default:
					t.Errorf("Unexpected HTTP method: %s", r.Method)
				}

			}))

			defer testServer.Close()
			cfg := &config.Config{
				BankSimURL:    testServer.URL,
				MaxRetryLimit: 3,
			}
			paymentID := uuid.New().String()
			httpClient := http.Client{}
			ctx := context.Background()
			logger := logrus.New().WithContext(ctx)
			retryHandler := clients.NewBankSimApiRetryHandler(httpClient, cfg, logger)

			response, err := retryHandler.BeginRetry(paymentID, cfg.BankSimURL+"/payments", &validRequest)

			if tt.succeedsAfterAttempts > 0 {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, response)
				assert.Equal(t, retryCounter, tt.succeedsAfterAttempts)
			}

			if tt.badRequestafterAttempts > 0 {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Equal(t, retryCounter, tt.badRequestafterAttempts)
			}
		})
	}
}

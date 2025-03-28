package clients_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/config"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/models"
)

var (
	validRequest = models.BankSimPaymentRequest{
		CardNumber: "123456789012345",
		ExpiryDate: "02/2090",
		Currency:   "EUR",
		Amount:     10,
		CVV:        "901",
	}
	invalidRequest = models.BankSimPaymentRequest{
		CardNumber: "123",
		ExpiryDate: "02/2000",
		Currency:   "ABC",
		Amount:     -1,
		CVV:        "901ssssss",
	}
)

func TestForwardPaymentRequest(t *testing.T) {
	tests := []struct {
		name             string
		request          *models.BankSimPaymentRequest
		shouldCallApi    bool
		expectedResponse *models.BankSimPaymentResponse
		expectedError    error
	}{
		{
			name:             "request is nil, returns error",
			request:          nil,
			shouldCallApi:    false,
			expectedResponse: nil,
			expectedError:    errors.New("bankSimPaymentRequest cannot be nil"),
		},
		{
			name:             "request triggers an error, returns error",
			request:          &invalidRequest,
			shouldCallApi:    true,
			expectedResponse: nil,
			expectedError:    errors.New("card number is invalid"),
		},
		{
			name:          "request resolves without error, returns response",
			request:       &validRequest,
			shouldCallApi: true,
			expectedResponse: &models.BankSimPaymentResponse{
				Authorized:        true,
				AuthorizationCode: "0bb07405-6d44-4b50-a14f-7ae0beff13ad",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calledServer := false

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "POST":
					calledServer = true
					assert.Equal(t, "", r.URL.RawQuery)
					req := models.BankSimPaymentRequest{}
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Errorf("error decoding request: %v", err)
						return
					}
					if req.CardNumber == validRequest.CardNumber {
						w.WriteHeader(http.StatusOK)
						m, err := json.Marshal(tt.expectedResponse)
						if err != nil {
							t.Errorf("error marshalling response: %v", err)
							return
						}
						w.Write(m)
					} else {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte(tt.expectedError.Error()))
					}

					break
				default:
					t.Errorf("Unexpected HTTP method: %s", r.Method)
				}

			}))

			defer testServer.Close()
			cfg := &config.Config{
				BankSimURL: testServer.URL,
			}

			httpClient := http.Client{}

			ctx := context.Background()
			logger := logrus.New().WithContext(ctx)
			bankSimApiClient := clients.NewBankSimAPI(httpClient, cfg, logger)

			response, err := bankSimApiClient.ForwardPaymentRequest(tt.request)

			if tt.expectedResponse != nil {
				assert.EqualValues(t, tt.expectedResponse, response)
			} else {
				assert.Nil(t, response)
			}

			if tt.expectedError != nil {
				assert.EqualValues(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.shouldCallApi, calledServer)

		})
	}
}

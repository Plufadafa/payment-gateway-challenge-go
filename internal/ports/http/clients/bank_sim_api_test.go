package clients_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients/mocks"
	"github.com/golang/mock/gomock"
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
	authorizedResponse = models.BankSimPaymentResponse{
		Authorized:        true,
		AuthorizationCode: "0bb07405-6d44-4b50-a14f-7ae0beff13ad",
	}
)

func TestForwardPaymentRequest(t *testing.T) {
	tests := []struct {
		name               string
		request            *models.BankSimPaymentRequest
		shouldTriggerRetry bool
		retryPayload       *models.BankSimPaymentResponse
		retryError         error
		shouldCallApi      bool
		expectedResponse   *models.BankSimPaymentResponse
		expectedError      error
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
			name:             "request resolves without error, returns response",
			request:          &validRequest,
			shouldCallApi:    true,
			expectedResponse: &authorizedResponse,
			expectedError:    nil,
		},
		{
			name:               "server unavailable and retry returns error",
			request:            &validRequest,
			shouldCallApi:      true,
			shouldTriggerRetry: true,
			retryError:         errors.New("retry returned an error"),
			retryPayload:       nil,
			expectedResponse:   nil,
			expectedError:      errors.New("retry returned an error"),
		},
		{
			name:               "server unavailable and retry succeeds",
			request:            &validRequest,
			shouldCallApi:      true,
			shouldTriggerRetry: true,
			retryError:         nil,
			retryPayload:       &authorizedResponse,
			expectedResponse:   &authorizedResponse,
			expectedError:      nil,
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
					if tt.shouldTriggerRetry {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}

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
						w.Write(m) //nolint errcheck
					} else {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte(tt.expectedError.Error())) //nolint errcheck
					}
				default:
					t.Errorf("Unexpected HTTP method: %s", r.Method)
				}

			}))

			defer testServer.Close()
			cfg := &config.Config{
				BankSimURL: testServer.URL,
			}
			paymentID := uuid.New().String()
			httpClient := http.Client{}
			ctx := context.Background()
			logger := logrus.New().WithContext(ctx)
			ctrl := gomock.NewController(t)
			mockRetryHandler := mocks.NewMockIBankSimApiRetryHandler(ctrl)
			bankSimApiClient := clients.NewBankSimAPI(httpClient, mockRetryHandler, cfg, logger)
			if tt.shouldTriggerRetry {
				mockRetryHandler.EXPECT().BeginRetry(paymentID, gomock.Eq(cfg.BankSimURL+"/payments"), gomock.Eq(tt.request)).Return(tt.retryPayload, tt.retryError).Times(1)
			} else {
				mockRetryHandler.EXPECT().BeginRetry(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			response, err := bankSimApiClient.ForwardPaymentRequest(paymentID, tt.request)

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

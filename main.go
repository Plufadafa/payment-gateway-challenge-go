package main

import (
	"context"
	"fmt"
	_ "github.com/cko-recruitment/payment-gateway-challenge-go/docs"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/config"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/helpers"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/clients"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/http/controllers"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/ports/repository"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

//	@title			Payment Gateway Challenge Go
//	@description	Interview challenge for building a Payment Gateway - Go version

//	@host		localhost:8090
//	@BasePath	/

// @securityDefinitions.basic	BasicAuth
func main() {
	ctx := context.Background()

	logger := logrus.New().WithContext(ctx)

	logger.Infof("version %s, commit %s, built at %s\n", version, commit, date)

	err := godotenv.Load("cmd/.env")
	if err != nil {
		logger.Fatalf("Error loading .env file")
	}

	var cfg config.Config
	err = envconfig.Process("", &cfg)
	if err != nil {
		logger.WithError(err).Fatal("error processing environment config")
	}

	port := ":8090"

	chiRouter := chi.NewRouter()
	chiRouter.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://localhost%s/swagger/doc.json", port)), //The url pointing to API definition
	))
	httpClient := http.Client{}

	paymentIdCreator := helpers.NewPaymentIDCreator()
	paymentsValidator := validation.NewPaymentValidator(logger)
	bankSimApiClient := clients.NewBankSimAPI(httpClient, &cfg, logger)
	paymentsRepository := repository.NewPaymentsRepository(logger)
	paymentsService := payments.NewService(paymentsRepository, bankSimApiClient, paymentsValidator, paymentIdCreator, logger)
	handlers := controllers.NewHandlers(paymentsService, paymentsValidator, paymentIdCreator, logger)
	handlers.SetupRoutes(chiRouter)

	logger.Fatal(http.ListenAndServe(port, chiRouter))
}

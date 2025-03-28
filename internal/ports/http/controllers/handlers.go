package controllers

import (
	"fmt"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/domain/payments"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/helpers"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/validation"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"net/http"
)

type (
	Handlers struct {
		paymentsService   payments.IService
		paymentsValidator validation.IPaymentValidator
		paymentIDCreator  helpers.IPaymentIDCreator
		logger            *logrus.Entry
	}

	IHandlers interface {
		SetupRoutes(chiRouter *chi.Mux)
	}
)

func NewHandlers(paymentsService payments.IService, paymentsValidator validation.IPaymentValidator, paymentIDCreator helpers.IPaymentIDCreator, logger *logrus.Entry) IHandlers {
	return &Handlers{
		paymentsService:   paymentsService,
		paymentsValidator: paymentsValidator,
		paymentIDCreator:  paymentIDCreator,
		logger:            logger,
	}
}

func (h *Handlers) SetupRoutes(chiRouter *chi.Mux) {
	chiRouter.HandleFunc(fmt.Sprintf("%s /ping", http.MethodGet), h.getPing)
	chiRouter.HandleFunc(fmt.Sprintf("%s /api/payments/{id}", http.MethodGet), h.getPayment)
	chiRouter.HandleFunc(fmt.Sprintf("%s /api/payments", http.MethodPost), h.processPayment)
}

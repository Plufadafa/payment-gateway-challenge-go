package repository

import (
	"errors"

	"github.com/sirupsen/logrus"
)

type (
	Payment struct {
		Id                 string `json:"id"`
		Authorized         string `json:"payment_status"`
		CardNumberLastFour int    `json:"card_number_last_four"`
		ExpiryMonth        int    `json:"expiry_month"`
		ExpiryYear         int    `json:"expiry_year"`
		Currency           string `json:"currency"`
		Amount             int    `json:"amount"`
	}
)

var (
	ErrPaymentIDCollision = errors.New("payment ID collision")
)

//go:generate mockgen -destination=./mocks/IPaymentsRepository.go -package mocks . IPaymentsRepository
type (
	PaymentsRepository struct {
		logger   *logrus.Entry
		payments map[string]Payment
	}

	IPaymentsRepository interface {
		GetPayment(id string) *Payment
		AddPayment(payment Payment) (*Payment, error)
	}
)

func NewPaymentsRepository(logger *logrus.Entry) IPaymentsRepository {
	return &PaymentsRepository{
		logger:   logger,
		payments: map[string]Payment{},
	}
}

func (ps *PaymentsRepository) GetPayment(id string) *Payment {
	p, ok := ps.payments[id]
	if !ok {
		return nil
	}
	return &p
}

func (ps *PaymentsRepository) AddPayment(payment Payment) (*Payment, error) {
	_, paymentExists := ps.payments[payment.Id]
	if paymentExists {
		ps.logger.Errorf("payment already exists with paymentID :[%s]", payment.Id)
		return nil, ErrPaymentIDCollision
	}

	ps.payments[payment.Id] = payment
	return &payment, nil
}

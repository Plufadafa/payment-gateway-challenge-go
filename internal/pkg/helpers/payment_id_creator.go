package helpers

import "github.com/google/uuid"

//go:generate mockgen -destination=./mocks/IPaymentIDCreator.go -package mocks . IPaymentIDCreator
type (
	PaymentIDCreator  struct{}
	IPaymentIDCreator interface {
		CreatePaymentID() string
	}
)

func NewPaymentIDCreator() IPaymentIDCreator {
	return &PaymentIDCreator{}
}

func (p *PaymentIDCreator) CreatePaymentID() string {
	return uuid.New().String()
}

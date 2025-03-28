package validation

import (
	"errors"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/pkg/models"
	"github.com/sirupsen/logrus"
	"regexp"
	"slices"
	"time"
)

//go:generate mockgen -destination=./mocks/IPaymentValidator.go -package mocks . IPaymentValidator
type (
	PaymentValidator struct {
		logger *logrus.Entry
	}

	IPaymentValidator interface {
		ValidateProcessPaymentRequest(paymentID string, request *models.ProcessPaymentRequest) error
	}
)

var (
	ErrInternal        = errors.New("internal error occurred")
	ErrCardExpired     = errors.New("card expired")
	acceptedCurrencies = []string{
		"EUR",
		"GBP",
		"YEN",
	}
)

const (
	cardNumberRegex = `^\d{14,18}$`
	cvvRegex        = `^\d{3,4}$`
)

func NewPaymentValidator(logger *logrus.Entry) IPaymentValidator {
	return &PaymentValidator{
		logger: logger,
	}
}

func (v *PaymentValidator) ValidateProcessPaymentRequest(paymentID string, request *models.ProcessPaymentRequest) error {
	if request == nil {
		return ErrInternal
	}
	if cardNumberErr := v.validateCardNumber(paymentID, request.CardNumber); cardNumberErr != nil {
		return cardNumberErr
	}

	if expiryMonthErr := v.validateExpiryMonth(request.ExpiryMonth); expiryMonthErr != nil {
		return expiryMonthErr
	}

	if expiryYearErr := v.validateExpiryYear(request.ExpiryMonth, request.ExpiryYear); expiryYearErr != nil {
		return expiryYearErr
	}

	if currencyErr := v.validateCurrencyCode(request.Currency); currencyErr != nil {
		return currencyErr
	}

	if amountErr := v.validateAmount(request.Amount); amountErr != nil {
		return amountErr
	}

	if cvvErr := v.validateCVV(paymentID, request.CVV); cvvErr != nil {
		return cvvErr
	}

	return nil
}

func (v *PaymentValidator) validateCardNumber(paymentID, cardNumber string) error {
	validMatch, err := regexp.Match(cardNumberRegex, []byte(cardNumber))
	if err != nil {
		v.logger.WithError(err).Errorf("error applying cardNumberRegex for paymentID: [%s]", paymentID)
		return ErrInternal
	}
	if !validMatch {
		return errors.New("invalid card number")
	}

	return nil
}

func (v *PaymentValidator) validateExpiryMonth(expiryMonth int) error {
	if expiryMonth < 1 || expiryMonth > 12 {
		return errors.New("expiryMonth must be between 1 and 12")
	}

	return nil
}

func (v *PaymentValidator) validateExpiryYear(expiryMonth int, expiryYear int) error {
	now := time.Now()
	nowYear := now.Year()
	nowMonth := int(now.Month())

	if expiryYear > nowYear {
		return nil
	}

	if expiryYear < nowYear {
		return ErrCardExpired
	}

	if expiryMonth <= nowMonth {
		return ErrCardExpired
	}

	return nil
}

func (v *PaymentValidator) validateCurrencyCode(currencyCode string) error {
	if slices.Contains(acceptedCurrencies, currencyCode) {
		return nil
	}

	return errors.New("currency not supported")
}

// assuming not supporting refunds + don't have a max transaction limit
func (v *PaymentValidator) validateAmount(amount int) error {
	if amount < 0 {
		return errors.New("amount must be greater than zero")
	}

	return nil
}

func (v *PaymentValidator) validateCVV(paymentID, cvv string) error {
	validMatch, err := regexp.Match(cvvRegex, []byte(cvv))
	if err != nil {
		v.logger.WithError(err).Errorf("error applying cvvRegex for paymentID: [%s]", paymentID)
		return ErrInternal
	}
	if !validMatch {
		return errors.New("invalid CVV")
	}

	return nil
}

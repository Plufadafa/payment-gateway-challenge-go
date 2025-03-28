package config

type Config struct {
	BankSimURL    string `required:"true" envconfig:"BANK_SIM_URL"`
	MaxRetryLimit int    `required:"true" envconfig:"MAX_RETRY_LIMIT"`
}

package config

type Config struct {
	BankSimURL string `required:"true" envconfig:"BANK_SIM_URL"`
}

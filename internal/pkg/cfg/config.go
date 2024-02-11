package cfg

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port                string
	GinMode             string
	GitHubQueryInterval string
}

func NewConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:                viper.GetString("PORT"),
		GinMode:             viper.GetString("GIN_MODE"),
		GitHubQueryInterval: viper.GetString("GITHUB_QUERY_INTERVAL"),
	}, nil
}

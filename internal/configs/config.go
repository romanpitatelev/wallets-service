package configs

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/rs/zerolog/log"
)

const envFileName = ".env"

type Config struct {
	env *EnvSetting
}

type EnvSetting struct {
	AppPort             int           `env:"APP_PORT" env-default:"8081" env-description:"Application port"`
	KafkaAddress        string        `env:"KAFKA_ADDRESS" env-default:"localhost:9094" env-description:"Kafka port"`
	PostgresDSN         string        `env:"POSTGRES_PORT" env-default:"postgresql://postgres:my_pass@localhost:5432/wallets_db" env-description:"PostgreSQL DSN"` //nolint:lll
	StaleWalletDuration time.Duration `env:"STALE_WALLET_DURATION" env-default:"24h" env-description:"The wallet is considered stale after this time duration"`
	PerformCheckPeriod  time.Duration `env:"PERFORM_CHECK_PERIOD" env-default:"1h" env-description:"Frequency of stale wallet checks"`
	XRServerAddress     string        `env:"XR_SERVER_ADDRESS" env-default:"http://localhost:2607" env-description:"XR server address"`
	XRgRPCServerAddress string        `env:"XR_GRPC_SERVER_ADDRESS" env-default:"http://localhost:2608" env-descritption:"XR gRPC server address"`
}

func findConfigFile() bool {
	_, err := os.Stat(envFileName)

	return err == nil
}

func (e *EnvSetting) GetHelpString() (string, error) {
	baseHeader := "Environment variables that can be set with env: "

	helpString, err := cleanenv.GetDescription(e, &baseHeader)
	if err != nil {
		return "", fmt.Errorf("failed to get help string: %w", err)
	}

	return helpString, nil
}

func New() *Config {
	envSetting := &EnvSetting{}

	helpString, err := envSetting.GetHelpString()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get help string")
	}

	log.Info().Msg(helpString)

	if findConfigFile() {
		if err := cleanenv.ReadConfig(envFileName, envSetting); err != nil {
			log.Panic().Err(err).Msg("failed to read env config")
		}
	} else if err := cleanenv.ReadEnv(envSetting); err != nil {
		log.Panic().Err(err).Msg("error reading env config")
	}

	return &Config{env: envSetting}
}

func (c *Config) PrintDebug() {
	envReflect := reflect.Indirect(reflect.ValueOf(c.env))
	envReflectType := envReflect.Type()

	exp := regexp.MustCompile("([Tt]oken|[Pp]assword)")

	for i := range envReflect.NumField() {
		key := envReflectType.Field(i).Name

		if exp.MatchString(key) {
			val, _ := envReflect.Field(i).Interface().(string)
			log.Debug().Msgf("%s: len %d", key, len(val))

			continue
		}

		log.Debug().Msgf("%s: %v", key, spew.Sprintf("%#v", envReflect.Field(i).Interface()))
	}
}

func (c *Config) GetAppPort() int {
	return c.env.AppPort
}

func (c *Config) GetKafkaAddress() string {
	return c.env.KafkaAddress
}

func (c *Config) GetPostgresDSN() string {
	return c.env.PostgresDSN
}

func (c *Config) GetStaleWalletDuration() time.Duration {
	return c.env.StaleWalletDuration
}

func (c *Config) GetPerformCheckPeriod() time.Duration {
	return c.env.PerformCheckPeriod
}

func (c *Config) GetXRHTTPServerAddress() string {
	return c.env.XRServerAddress
}

func (c *Config) GetXRgRPCServerAddress() string {
	return strings.TrimPrefix(c.env.XRgRPCServerAddress, "http://")
}

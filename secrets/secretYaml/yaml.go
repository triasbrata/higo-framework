package secretyaml

import (
	"time"

	"github.com/spf13/viper"
	"github.com/triasbrata/higo/secrets"
	"github.com/triasbrata/higo/secrets/utils"
)

type secret struct {
	engine *viper.Viper
}

// GetSecretAsBool implements secrets.Secret.
func (s *secret) GetSecretAsBool(key string, defaultValue bool) bool {
	return utils.GetSecretAsBool(s.engine, key, defaultValue)
}

// GetSecretAsDuration implements secrets.Secret.
func (s *secret) GetSecretAsDuration(key string, defaultValue time.Duration) time.Duration {
	return utils.GetSecretAsDuration(s.engine, key, defaultValue)
}

// GetSecretAsFloat64 implements secrets.Secret.
func (s *secret) GetSecretAsFloat64(key string, defaultValue float64) float64 {
	return utils.GetSecretAsFloat64(s.engine, key, defaultValue)

}

// GetSecretAsInt64 implements secrets.Secret.
func (s *secret) GetSecretAsInt64(key string, defaultValue int64) int64 {
	return utils.GetSecretAsInt64(s.engine, key, defaultValue)
}

// GetSecretAsMap implements secrets.Secret.
func (s *secret) GetSecretAsMap(key string, defaultValue map[string]string) map[string]string {
	return utils.GetSecretAsMap(s.engine, key, defaultValue)
}

// GetSecretAsSlice implements secrets.Secret.
func (s *secret) GetSecretAsSlice(key string, defaultValue []string) []string {
	return utils.GetSecretAsSlice(s.engine, key, defaultValue)
}

// GetSecretAsSliceOfInt64 implements secrets.Secret.
func (s *secret) GetSecretAsSliceOfInt64(key string, defaultValue []int64) []int64 {
	return utils.GetSecretAsSliceOfInt64(s.engine, key, defaultValue)
}

// GetSecretAsString implements secrets.Secret.
func (s *secret) GetSecretAsString(key string, defaultValue string) string {
	return utils.GetSecretAsString(s.engine, key, defaultValue)
}

func NewSecretFromYaml() (secrets.Secret, error) {
	engine := viper.New()
	err := utils.NewEngineViper("yaml", "yml", engine)
	if err != nil {
		return nil, err
	}
	return &secret{
		engine: engine,
	}, nil
}

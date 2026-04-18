package secretjson

import (
	"time"

	"github.com/spf13/viper"
	"github.com/triasbrata/higo/secrets"
	"github.com/triasbrata/higo/secrets/utils"
)

type secJson struct {
	engine *viper.Viper
}

// GetSecretAsBool implements secrets.Secret.
func (s *secJson) GetSecretAsBool(key string, defaultValue bool) bool {
	return utils.GetSecretAsBool(s.engine, key, defaultValue)
}

// GetSecretAsDuration implements secrets.Secret.
func (s *secJson) GetSecretAsDuration(key string, defaultValue time.Duration) time.Duration {
	return utils.GetSecretAsDuration(s.engine, key, defaultValue)
}

// GetSecretAsFloat64 implements secrets.Secret.
func (s *secJson) GetSecretAsFloat64(key string, defaultValue float64) float64 {
	return utils.GetSecretAsFloat64(s.engine, key, defaultValue)

}

// GetSecretAsInt64 implements secrets.Secret.
func (s *secJson) GetSecretAsInt64(key string, defaultValue int64) int64 {
	return utils.GetSecretAsInt64(s.engine, key, defaultValue)
}

// GetSecretAsMap implements secrets.Secret.
func (s *secJson) GetSecretAsMap(key string, defaultValue map[string]string) map[string]string {
	return utils.GetSecretAsMap(s.engine, key, defaultValue)
}

// GetSecretAsSlice implements secrets.Secret.
func (s *secJson) GetSecretAsSlice(key string, defaultValue []string) []string {
	return utils.GetSecretAsSlice(s.engine, key, defaultValue)
}

// GetSecretAsSliceOfInt64 implements secrets.Secret.
func (s *secJson) GetSecretAsSliceOfInt64(key string, defaultValue []int64) []int64 {
	return utils.GetSecretAsSliceOfInt64(s.engine, key, defaultValue)
}

// GetSecretAsString implements secrets.Secret.
func (s *secJson) GetSecretAsString(key string, defaultValue string) string {
	return utils.GetSecretAsString(s.engine, key, defaultValue)
}

func NewSecretJson() (secrets.Secret, error) {
	engine := viper.NewWithOptions(viper.WithCodecRegistry(newSonicCodec()))
	utils.NewEngineViper("json", "json", engine)
	return &secJson{engine: engine}, nil
}

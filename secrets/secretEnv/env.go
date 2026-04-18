package secretEnv

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/triasbrata/higo/secrets"
)

type secretEnv struct {
}

func lookupEnv(key string) (string, bool) {
	value, exists := os.LookupEnv(key)
	if exists {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = strings.TrimLeft(strings.TrimRight(value, `"`), `"`)
		}
		return value, exists
	}
	return "", exists
}

// GetSecretAsBool implements secrets.Secret.
func (s *secretEnv) GetSecretAsBool(key string, defaultValue bool) bool {
	lookupVal, exists := lookupEnv(key)
	val, err := strconv.ParseBool(lookupVal)
	if !exists {
		return defaultValue
	}
	if err != nil {
		slog.Warn("Parse secret bool", slog.Any("err", err))
		return defaultValue
	}
	return val
}

// GetSecretAsDuration implements secrets.Secret.
func (s *secretEnv) GetSecretAsDuration(key string, defaultValue time.Duration) time.Duration {
	lookupVal, exists := lookupEnv(key)
	val, err := time.ParseDuration(lookupVal)
	if !exists {
		return defaultValue
	}
	if err != nil {
		slog.Warn("Parse secret bool", slog.Any("err", err))
		return defaultValue
	}
	return val
}

// GetSecretAsFloat64 implements secrets.Secret.
func (s *secretEnv) GetSecretAsFloat64(key string, defaultValue float64) float64 {
	lookupVal, exists := lookupEnv(key)
	val, err := strconv.ParseFloat(lookupVal, 64)
	if !exists {
		return defaultValue
	}
	if err != nil {
		slog.Warn("Parse secret bool", slog.Any("err", err))
		return defaultValue
	}
	return val
}

// GetSecretAsInt64 implements secrets.Secret.
func (s *secretEnv) GetSecretAsInt64(key string, defaultValue int64) int64 {
	lookupVal, exists := lookupEnv(key)
	val, err := strconv.ParseInt(lookupVal, 10, 64)
	if !exists {
		return defaultValue
	}
	if err != nil {
		slog.Warn("Parse secret bool", slog.Any("err", err))
		return defaultValue
	}
	return val
}

// GetSecretAsMap implements secrets.Secret.
func (s *secretEnv) GetSecretAsMap(key string, defaultValue map[string]string) map[string]string {
	lookupVal, exists := lookupEnv(key)
	if !exists {
		return defaultValue
	}
	split := strings.Split(lookupVal, ",")
	out := make(map[string]string)
	for _, item := range split {
		dsplit := strings.Split(item, ":")
		if len(dsplit) == 0 {
			continue
		}
		if len(dsplit) == 1 {
			out[dsplit[0]] = ""
		}
		if len(dsplit) > 1 {
			out[dsplit[0]] = dsplit[1]
		}
	}
	return out
}

// GetSecretAsSlice implements secrets.Secret.
func (s *secretEnv) GetSecretAsSlice(key string, defaultValue []string) []string {
	lookupVal, exists := lookupEnv(key)
	if !exists {
		return defaultValue
	}
	split := strings.Split(lookupVal, ",")
	return split
}

// GetSecretAsSliceOfInt64 implements secrets.Secret.
func (s *secretEnv) GetSecretAsSliceOfInt64(key string, defaultValue []int64) []int64 {
	lookupVal, exists := lookupEnv(key)
	if !exists {
		return defaultValue
	}
	split := strings.Split(lookupVal, ",")
	out := make([]int64, 0, len(split))
	for _, item := range split {
		val, err := strconv.ParseInt(lookupVal, 10, 64)
		if err != nil {
			slog.Warn("Parse secret GetSecretAsSliceOfInt64", slog.Any("err", err), slog.Any("item", item))
			continue
		}
		out = append(out, val)
	}
	return out
}

// GetSecretAsString implements secrets.Secret.
func (s *secretEnv) GetSecretAsString(key string, defaultValue string) string {
	lookupVal, exists := lookupEnv(key)
	if !exists {
		return defaultValue
	}
	return lookupVal
}

func NewSecretFromEnv() (secrets.Secret, error) {
	err := dotenv.Load()
	if err != nil {
		return nil, err
	}
	return &secretEnv{}, nil
}

package utils

import (
	"flag"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewEngineViper(format string, ext string, engine *viper.Viper) error {
	flag.String("secretPath", "", fmt.Sprintf("secret path. \n\tex:~/.config/config.%s", ext))
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	if err := engine.BindPFlags(pflag.CommandLine); err != nil {
		return fmt.Errorf("failed bind flag %w", err)
	}
	secretPath := engine.GetString("secretPath")
	if secretPath == "" {
		return fmt.Errorf("cant find secret file while secretPath was empty")
	}
	engine.SetConfigType("yaml")
	engine.SetConfigFile(secretPath)

	if err := engine.ReadInConfig(); err != nil {
		return fmt.Errorf("fail read %s config %w", format, err)
	}
	return nil
}

// GetSecretAsBool implements secrets.Secret.
func GetSecretAsBool(s *viper.Viper, key string, defaultValue bool) bool {
	s.SetDefault(key, defaultValue)
	return s.GetBool(key)
}

// GetSecretAsDuration implements secrets.Secret.
func GetSecretAsDuration(s *viper.Viper, key string, defaultValue time.Duration) time.Duration {
	s.SetDefault(key, defaultValue)
	return s.GetDuration(key)
}

// GetSecretAsFloat64 implements secrets.Secret.
func GetSecretAsFloat64(s *viper.Viper, key string, defaultValue float64) float64 {
	s.SetDefault(key, defaultValue)
	return s.GetFloat64(key)
}

// GetSecretAsInt64 implements secrets.Secret.
func GetSecretAsInt64(s *viper.Viper, key string, defaultValue int64) int64 {
	s.SetDefault(key, defaultValue)
	return s.GetInt64(key)
}

// GetSecretAsMap implements secrets.Secret.
func GetSecretAsMap(s *viper.Viper, key string, defaultValue map[string]string) map[string]string {
	s.SetDefault(key, defaultValue)
	return s.GetStringMapString(key)
}

// GetSecretAsSlice implements secrets.Secret.
func GetSecretAsSlice(s *viper.Viper, key string, defaultValue []string) []string {
	s.SetDefault(key, defaultValue)
	return s.GetStringSlice(key)
}

// GetSecretAsSliceOfInt64 implements secrets.Secret.
func GetSecretAsSliceOfInt64(s *viper.Viper, key string, defaultValue []int64) []int64 {
	val := s.Get(key)
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || rv.IsZero() || rv.IsNil() {
		return defaultValue
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]int64, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			rawValue := rv.Index(i).Interface()
			casted, err := strconv.ParseInt(fmt.Sprintf("%v", rawValue), 10, 64)
			if err != nil {
				slog.Warn("fail casting at GetSecretAsSliceOfInt64", slog.Any("err", err), slog.Any("value", rawValue))
				continue
			}
			out = append(out, casted)
		}
		return out
	}
	return defaultValue
}

// GetSecretAsString implements secrets.Secret.
func GetSecretAsString(s *viper.Viper, key string, defaultValue string) string {
	s.SetDefault(key, defaultValue)
	return s.GetString(key)
}

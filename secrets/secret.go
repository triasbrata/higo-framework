package secrets

import "time"

type Secret interface {
	GetSecretAsInt64(key string, defaultValue int64) int64
	GetSecretAsFloat64(key string, defaultValue float64) float64
	GetSecretAsBool(key string, defaultValue bool) bool
	GetSecretAsString(key string, defaultValue string) string
	GetSecretAsDuration(key string, defaultValue time.Duration) time.Duration
	GetSecretAsSliceOfInt64(key string, defaultValue []int64) []int64
	GetSecretAsSlice(key string, defaultValue []string) []string
	GetSecretAsMap(key string, defaultValue map[string]string) map[string]string
}

package secretjson

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/spf13/viper"
)

type sonicCodec struct {
}

// Encode implements viper.Encoder.
func (s *sonicCodec) Encode(v map[string]any) ([]byte, error) {
	return sonic.Marshal(v)
}

// Decode implements viper.Decoder.
func (s *sonicCodec) Decode(b []byte, v map[string]any) error {
	return sonic.Unmarshal(b, v)
}

// Decoder implements viper.CodecRegistry.
func (s *sonicCodec) Decoder(format string) (viper.Decoder, error) {
	if format == "json" {
		return s, nil
	}
	return nil, fmt.Errorf("only can handle json codec")
}

// Encoder implements viper.CodecRegistry.
func (s *sonicCodec) Encoder(format string) (viper.Encoder, error) {
	if format == "json" {
		return s, nil
	}
	return nil, fmt.Errorf("only can handle json codec")
}

func newSonicCodec() viper.CodecRegistry {
	return &sonicCodec{}
}

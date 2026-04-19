package log

import (
	"log/slog"
	"os"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type LogConfig struct {
	AddSource bool
	Level     slog.Level
}

// LoadLoggerSlog registers an slog.Logger with fx.
// If no LogConfig is provided, LOG_ADD_SOURCE and LOG_LEVEL env vars are used.
func LoadLoggerSlog(cfg ...LogConfig) fx.Option {
	var c LogConfig
	if len(cfg) > 0 {
		c = cfg[0]
	} else {
		c.AddSource = strings.EqualFold(os.Getenv("LOG_ADD_SOURCE"), "true")
		c.Level = parseLevel(os.Getenv("LOG_LEVEL"))
	}
	return fx.Options(
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
		fx.Provide(func() *slog.Logger {
			handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: c.AddSource,
				Level:     c.Level,
			})
			logger := slog.New(handler)
			slog.SetDefault(logger)
			return logger
		}),
	)
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	_ = l.UnmarshalText([]byte(s))
	return l
}

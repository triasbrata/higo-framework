package log

import (
	"log/slog"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func LoadLoggerSlog() fx.Option {
	return fx.Options(
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{
				Logger: logger,
			}
		}),
		fx.Provide(func() *slog.Logger {
			handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
			logger := slog.New(handler)
			slog.SetDefault(logger)
			return logger
		}),
	)
}

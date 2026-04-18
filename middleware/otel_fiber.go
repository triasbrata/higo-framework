package middleware

import (
	"fmt"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/triasbrata/higo/routers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func OtelFiberGlobal(router routers.Router, tp trace.TracerProvider, mp metric.MeterProvider) {
	router.GlobalMiddleware(otelfiber.Middleware(
		otelfiber.WithCollectClientIP(true),
		otelfiber.WithTracerProvider(tp),
		otelfiber.WithMeterProvider(mp),
		otelfiber.WithPropagators(otel.GetTextMapPropagator()),
		otelfiber.WithSpanNameFormatter(func(ctx *fiber.Ctx) string {
			pattern := ctx.Route().Path
			if pattern == "" {
				pattern = ctx.OriginalURL()
			}
			return fmt.Sprintf("%s %s", ctx.Method(), pattern)
		}),
	))
}

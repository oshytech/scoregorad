package observability

import (
	"context"
	"log/slog"
	"os"
)

type loggerKey struct{}

// NewLogger crea un *slog.Logger configurado para el entorno.
// En producción usa JSON (parseble por Datadog, Loki, CloudWatch).
// En desarrollo usa text (legible por humanos en terminal).
//
// slog es parte de la stdlib desde Go 1.21 — no necesitamos zap ni zerolog
// para un caso de uso estándar. La decisión de no añadir dependencia es
// explícita y merece un párrafo en el artículo.
func NewLogger(env string) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// WithLogger almacena el logger en el context de la petición.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext extrae el logger del context.
// Si no hay logger en el context, devuelve el logger por defecto.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

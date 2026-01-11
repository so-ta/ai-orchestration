package telemetry

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPMiddleware returns an HTTP middleware for tracing
func HTTPMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "http.request",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

// WrapHTTPClient wraps an HTTP client with tracing
func WrapHTTPClient(client *http.Client) *http.Client {
	client.Transport = otelhttp.NewTransport(client.Transport)
	return client
}

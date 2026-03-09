package middleware

import (
	"log"
	"net/http"
	"time"
)

func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("%s %s %s", colorMethod(r.Method), r.URL.Path, colorDuration(duration))
	})
}

func colorMethod(method string) string {
	switch method {
	case http.MethodGet:
		return "\033[32m" + method + "\033[0m" // green
	case http.MethodPost:
		return "\033[33m" + method + "\033[0m" // yellow
	case http.MethodDelete:
		return "\033[31m" + method + "\033[0m" // red
	default:
		return "\033[36m" + method + "\033[0m" // cyan
	}
}

func colorDuration(d time.Duration) string {
	color := ""
	switch {
	case d < 100*time.Millisecond:
		color = "\033[32m" // green
	case d < 500*time.Millisecond:
		color = "\033[33m" // yellow
	default:
		color = "\033[31m" // red
	}
	return color + d.String() + "\033[0m"
}

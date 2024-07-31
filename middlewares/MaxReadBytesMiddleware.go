package middlewares

import (
	"holvit/config"
	"net/http"
)

func MaxReadBytesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.C.Server.MaxReadBytes)
		next.ServeHTTP(w, r)
	})
}

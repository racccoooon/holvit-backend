package middlewares

import (
	"holvit/logging"
	"net/http"
)

func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logging.Logger.Infof("[%s] %v", r.Method, r.URL)

		next.ServeHTTP(w, r)
	})
}

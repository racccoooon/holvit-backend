package middlewares

import (
	"holvit/config"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/requestContext"
	"net/http"
	"runtime/debug"
)

func ErrorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := GetScope(r.Context())
		rcs := ioc.Get[requestContext.RequestContextService](scope)

		defer func() {
			if err := recover(); err != nil {
				// Log the panic and stack trace
				logging.Logger.Errorf("panic: %v\n%s", err, debug.Stack())

				// Return a 500 Internal Server Error response
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)

		errors := rcs.Errors()
		if len(errors) != 0 {
			err := errors[0]
			switch err.(type) {
			case *httpErrors.HttpError:
				httpErr := err.(*httpErrors.HttpError)

				message := httpErr.Message()
				if config.C.IsProduction() && httpErr.Status() == http.StatusUnauthorized {
					message = ""
				}

				logging.Logger.Error(err)
				http.Error(w, message, httpErr.Status())
				break

			default:
				msg := "An internal server error occurred"

				if config.C.IsDevelopment() {
					msg = err.Error()
				}

				logging.Logger.Error(err)
				http.Error(w, msg, http.StatusInternalServerError)
				break
			}

			return
		}
	})
}

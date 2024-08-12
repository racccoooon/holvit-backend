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

func handleError(w http.ResponseWriter, err error) {
	switch err := err.(type) {
	case *httpErrors.HttpError:
		message := err.Message()
		if config.C.IsProduction() && err.Status() == http.StatusUnauthorized {
			message = ""
		}

		logging.Logger.Info(err)
		http.Error(w, message, err.Status())
	default:
		msg := "An internal server error occurred"

		if config.C.IsDevelopment() {
			msg = err.Error()
		}

		logging.Logger.Error(err)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func ErrorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := GetScope(r.Context())
		rcs := ioc.Get[requestContext.RequestContextService](scope)

		defer func() {
			if err := recover(); err != nil {
				logging.Logger.Errorf("panic: %v\n%s", err, debug.Stack())
				if e, ok := err.(error); ok {
					handleError(w, e)
				}
			}
		}()
		next.ServeHTTP(w, r)

		errors := rcs.Errors()
		if len(errors) != 0 {
			err := errors[0]
			handleError(w, err)
		}
	})
}

package middlewares

import (
	"holvit/config"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/requestContext"
	"net/http"
)

func ErrorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := GetScope(r.Context())
		rcs := ioc.Get[requestContext.RequestContextService](scope)

		next.ServeHTTP(w, r)

		errors := rcs.Errors()
		if len(errors) != 0 {
			err := errors[0]
			switch err.(type) {
			case *httpErrors.HttpError:
				httpErr := err.(*httpErrors.HttpError)
				http.Error(w, httpErr.Message(), httpErr.Status())
				break

			default:
				msg := "An internal server error occurred"

				if config.C.IsDevelopment() {
					msg = err.Error()
				}

				http.Error(w, msg, http.StatusInternalServerError)
				break
			}

			return
		}
	})
}

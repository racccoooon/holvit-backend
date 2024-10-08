package middlewares

import (
	"context"
	"github.com/gorilla/mux"
	"holvit/ioc"
	"holvit/utils"
	"net/http"
)

func ScopeMiddleware(dp *ioc.DependencyProvider) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := dp.NewScope()
			defer utils.PanicOnErr(scope.Close)

			r = r.WithContext(ContextWithNewScope(r.Context(), scope))
			next.ServeHTTP(w, r)
		})
	}
}

type scopeKeyType int

const scopeKey scopeKeyType = iota

func ContextWithNewScope(ctx context.Context, scope *ioc.DependencyProvider) context.Context {
	return context.WithValue(ctx, scopeKey, scope)
}

func GetScope(ctx context.Context) *ioc.DependencyProvider {
	return ctx.Value(scopeKey).(*ioc.DependencyProvider)
}

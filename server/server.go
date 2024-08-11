package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"holvit/config"
	"holvit/handlers"
	"holvit/handlers/auth"
	"holvit/handlers/oidc"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/routes"
	"holvit/services"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func Serve(dp *ioc.DependencyProvider) {
	address := fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	logging.Logger.Infof("Serving api and frontend on %s", address)

	r := mux.NewRouter()

	r.Use(middlewares.AccessLogMiddleware)
	r.Use(middlewares.MaxReadBytesMiddleware)

	r.Use(middlewares.ScopeMiddleware(dp))
	r.Use(middlewares.ErrorHandlingMiddleware)

	r.Use(services.CurrentSessionMiddleware)

	r.HandleFunc(routes.AdminFrontend.String(), func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, routes.AdminFrontend.String()+"/", http.StatusFound)
	}).Methods("GET")

	r.MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		return strings.HasPrefix(request.URL.Path, routes.AdminFrontend.String()+"/")
	}).HandlerFunc(handlers.AdminFrontend)

	r.HandleFunc(routes.ApiHealth.String(), handlers.Health).Methods("GET")

	r.HandleFunc(routes.OidcAuthorize.String(), oidc.Authorize).Methods("GET", "POST")
	r.HandleFunc(routes.OidcToken.String(), oidc.Token).Methods("POST")
	r.HandleFunc(routes.OidcUserInfo.String(), oidc.UserInfo).Methods("GET", "POST")
	r.HandleFunc(routes.OidcJwks.String(), oidc.Jwks)
	r.HandleFunc(routes.OidcLogout.String(), oidc.EndSession)
	r.HandleFunc(routes.WellKnown.String(), oidc.WellKnown)

	r.HandleFunc(routes.ApiVerifyPassword.String(), auth.VerifyPassword).Methods("POST")
	r.HandleFunc(routes.ApiResetPassword.String(), auth.ResetPassword).Methods("POST")
	r.HandleFunc(routes.ApiTotpOnboarding.String(), auth.TotpOnboarding).Methods("POST")
	r.HandleFunc(routes.ApiVerifyTotp.String(), auth.VerifyTotp).Methods("POST")
	r.HandleFunc(routes.ApiVerifyDevice.String(), auth.VerifyDevice).Methods("POST")
	r.HandleFunc(routes.ApiGetOnboardingTotp.String(), auth.GetOnboardingTotp).Methods("POST")

	r.HandleFunc(routes.AuthorizeGrant.String(), auth.AuthorizeGrant).Methods("POST")
	r.HandleFunc(routes.AuthVerifyEmail.String(), auth.VerifyEmail).Methods("GET")
	r.HandleFunc(routes.LoginComplete.String(), auth.CompleteAuthFlow).Methods("POST")
	//TODO: r.HandleFunc(routes.ApiResendEmailVerification.String(), auth.ResendEmailVerification).Methods("POST")

	registerStatics(r)

	srv := &http.Server{
		Handler:      r,
		Addr:         address,
		WriteTimeout: config.C.Server.WriteTimeout,
		ReadTimeout:  config.C.Server.ReadTimeout,
	}

	go serve(srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), config.C.Server.ShutdownTimeout)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		panic(err)
	}
}

func serve(srv *http.Server) {
	if err := srv.ListenAndServe(); err != nil {
		logging.Logger.Fatalf("Failed to serve api: %v", err)
	}
}

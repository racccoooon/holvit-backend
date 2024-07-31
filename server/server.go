package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"holvit/config"
	"holvit/handlers"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func ServeApi(dp *ioc.DependencyProvider) {
	address := fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	logging.Logger.Infof("Serving api on %s", address)

	r := mux.NewRouter()

	r.Use(middlewares.ScopeMiddleware(dp))
	r.Use(middlewares.ErrorHandlingMiddleware)

	r.Use(middlewares.AccessLogMiddleware)

	r.Use(middlewares.MaxReadBytesMiddleware)
	// r.Use(middlewares.EnforceJsonMiddleware)

	r.HandleFunc("/api/health", handlers.Health).Methods("GET")

	r.HandleFunc("/oidc/authorize/{realmName}", handlers.Authorize).Methods("GET", "POST")
	r.HandleFunc("/oidc/authorize-grant", handlers.AuthorizeGrant).Methods("POST")

	r.HandleFunc("/oidc/token", handlers.Token)
	r.HandleFunc("/oidc/userinfo", handlers.Token)
	r.HandleFunc("/oidc/jwks", handlers.Token)
	r.HandleFunc("/oidc/logout", handlers.Token)

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

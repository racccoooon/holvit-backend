package handlers

import (
	"github.com/gorilla/mux"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/services"
	"net/http"
	"strings"
)

func Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	routeParams := mux.Vars(r)

	request := services.AuthorizationRequest{
		ResponseTypes: strings.Split(r.Form.Get("response_type"), " "),
		RealmName:     routeParams["realmName"],
		ClientId:      r.Form.Get("client_id"),
		RedirectUri:   r.Form.Get("redirect_uri"),
		Scopes:        strings.Split(r.Form.Get("scope"), " "),
		State:         r.Form.Get("state"),
		ResponseMode:  r.Form.Get("response_mode"),
	}

	oidcService := ioc.Get[services.OidcService](scope)
	response, err := oidcService.Authorize(ctx, request)
	if err != nil {
		rcs.Error(err)
		return
	}

	response.HandleHttp(w, r)
}

func Token(w http.ResponseWriter, r *http.Request) {
}

func UserInfo(w http.ResponseWriter, r *http.Request) {
}

func Jwks(w http.ResponseWriter, r *http.Request) {
}

func EndSession(w http.ResponseWriter, r *http.Request) {
}

func Discovery(w http.ResponseWriter, r *http.Request) {
}

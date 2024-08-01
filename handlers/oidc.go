package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"holvit/constants"
	"holvit/httpErrors"
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

func AuthorizeGrant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	grants := r.Form["grant"]
	token := r.Form.Get("token")

	if len(grants) == 0 {
		rcs.Error(httpErrors.BadRequest().WithMessage("Missing grants"))
		return
	}
	if token == "" {
		rcs.Error(httpErrors.BadRequest().WithMessage("Missing token"))
		return
	}

	tokenService := ioc.Get[services.TokenService](scope)
	info, err := tokenService.RetrieveGrantInfo(ctx, token)
	if err != nil {
		rcs.Error(err)
		return
	}

	oidcService := ioc.Get[services.OidcService](scope)
	response, err := oidcService.Grant(ctx, services.GrantRequest{
		ClientId:             info.ClientId,
		RealmId:              info.RealmId,
		ScopeNames:           grants,
		AuthorizationRequest: info.AuthorizationRequest,
	})
	if err != nil {
		rcs.Error(err)
		return
	}

	response.HandleHttp(w, r)
}

func Token(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	grantType := r.Form.Get("grant_type")

	clientId, clientSecret, _ := r.BasicAuth()

	oidcService := ioc.Get[services.OidcService](scope)

	var response *services.TokenResponse
	var err error

	switch grantType {
	case constants.TokenGrantTypeAuthorizationCode:
		response, err = oidcService.HandleAuthorizationCode(ctx, services.AuthorizationCodeTokenRequest{
			RedirectUri:  r.Form.Get("redirect_uri"),
			Code:         r.Form.Get("code"),
			ClientId:     clientId,
			ClientSecret: clientSecret,
		})
		break
	case constants.TokenGrantTypeRefreshToken:
		response, err = oidcService.HandleRefreshToken(ctx, services.RefreshTokenRequest{
			RefreshToken: r.Form.Get("refresh_token"),
			ClientId:     clientId,
			ClientSecret: clientSecret,
		})
		break
	default:
		rcs.Error(httpErrors.BadRequest().WithMessage(fmt.Sprintf("Unsupported grant_type '%s'", grantType)))
	}

	if err != nil {
		rcs.Error(err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		rcs.Error(err)
		return
	}

}

func UserInfo(w http.ResponseWriter, r *http.Request) {
}

func Jwks(w http.ResponseWriter, r *http.Request) {
}

func EndSession(w http.ResponseWriter, r *http.Request) {
}

func Discovery(w http.ResponseWriter, r *http.Request) {
}

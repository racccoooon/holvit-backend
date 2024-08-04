package oidc

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/requestContext"
	"holvit/services"
	"holvit/utils"
	"net/http"
	"strings"
)

func login(w http.ResponseWriter, r *http.Request, realmName string, request services.AuthorizationRequest) error {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	if err := r.ParseForm(); err != nil {
		return err
	}

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realms, _, err := realmRepository.FindRealms(ctx, repositories.RealmFilter{
		Name: &realmName,
	})
	if err != nil {
		return err
	}
	realm := realms[0]

	tokenService := ioc.Get[services.TokenService](scope)
	loginToken, err := tokenService.StoreLoginCode(ctx, services.LoginInfo{
		NextStep: constants.AuthenticateStepVerifyPassword,
		RealmId:  realm.Id,
		Request:  request,
	})
	if err != nil {
		return err
	}

	frontendData := utils.AuthFrontendData{
		Mode: constants.FrontendModeAuthenticate,
		Authenticate: &utils.AuthFrontendDataAuthenticate{
			Token:         loginToken,
			UseRememberMe: realm.EnableRememberMe,
		},
	}

	err = utils.ServeAuthFrontend(w, frontendData)
	if err != nil {
		return err
	}

	return nil
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	routeParams := mux.Vars(r)
	realmName := routeParams["realmName"]

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}
	request := services.AuthorizationRequest{
		ResponseTypes: strings.Split(r.Form.Get("response_type"), " "),
		RealmName:     realmName,
		ClientId:      r.Form.Get("client_id"),
		RedirectUri:   r.Form.Get("redirect_uri"),
		Scopes:        strings.Split(r.Form.Get("scope"), " "),
		State:         r.Form.Get("state"),
		ResponseMode:  r.Form.Get("response_mode"),
	}

	currentUserService := ioc.Get[services.CurrentUserService](scope)

	if err := currentUserService.VerifyAuthorized(); err == nil {
		err := login(w, r, realmName, request)
		if err != nil {
			rcs.Error(err)
			return
		}
		return
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
			ScopeNames:   strings.Split(r.Form.Get("scope"), " "),
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
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	bearer := r.Header.Get("Authorization")

	oidcService := ioc.Get[services.OidcService](scope)
	response := oidcService.UserInfo(bearer)

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		return
	}
}

func Jwks(w http.ResponseWriter, r *http.Request) {
}

func EndSession(w http.ResponseWriter, r *http.Request) {
}

func Discovery(w http.ResponseWriter, r *http.Request) {
}

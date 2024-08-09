package oidc

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"holvit/constants"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
	"holvit/routes"
	"holvit/services"
	"net/http"
	"strings"
)

func login(w http.ResponseWriter, r *http.Request, realmName string) error {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	if err := r.ParseForm(); err != nil {
		return err
	}

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealms(ctx, repos.RealmFilter{
		Name: h.Some(realmName),
	}).First()

	tokenService := ioc.Get[services.TokenService](scope)
	loginToken := tokenService.StoreLoginCode(ctx, services.LoginInfo{
		NextStep: constants.AuthenticateStepVerifyPassword,
		RealmId:  realm.Id,
		// TODO: the original url thing does not work if the initial request was a POST request -- how to deal with that?
		OriginalUrl: r.URL.String(),
	})

	frontendData := services.AuthFrontendData{
		Mode: constants.FrontendModeAuthenticate,
		Authenticate: &services.AuthFrontendDataAuthenticate{
			ClientName:       "TODO (client name)",
			Token:            loginToken,
			UseRememberMe:    realm.EnableRememberMe,
			RegisterUrl:      "TODO (register URL)",
			LoginCompleteUrl: routes.LoginComplete.Url(realmName),
		},
	}

	frontendService := ioc.Get[services.FrontendService](scope)

	frontendService.WriteAuthFrontend(w, realmName, frontendData)
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
		ResponseTypes:       strings.Split(r.Form.Get("response_type"), " "),
		RealmName:           realmName,
		ClientId:            r.Form.Get("client_id"),
		RedirectUri:         r.Form.Get("redirect_uri"),
		Scopes:              strings.Split(r.Form.Get("scope"), " "),
		State:               r.Form.Get("state"),
		ResponseMode:        r.Form.Get("response_mode"),
		PKCEChallenge:       r.Form.Get("code_challenge"),
		PKCEChallengeMethod: r.Form.Get("code_challenge_method"),
	}

	currentUserService := ioc.Get[services.CurrentSessionService](scope)

	if err := currentUserService.VerifyAuthorized(); err != nil {
		err := login(w, r, realmName)
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

	clientId, clientSecretStr, hasBasicAuth := r.BasicAuth()

	clientSecret := h.None[string]()
	if hasBasicAuth {
		clientSecret = h.Some(clientSecretStr)
	} else {
		clientId = r.Form.Get("client_id")
	}

	pkceVerifierStr := r.Form.Get("code_verifier")
	pkceVerifier := h.None[string]()
	if pkceVerifierStr != "" {
		pkceVerifier = h.Some(pkceVerifierStr)
	}

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
			PKCEVerifier: pkceVerifier,
		})
	case constants.TokenGrantTypeRefreshToken:
		response, err = oidcService.HandleRefreshToken(ctx, services.RefreshTokenRequest{
			RefreshToken: r.Form.Get("refresh_token"),
			ClientId:     clientId,
			ClientSecret: clientSecret,
			ScopeNames:   strings.Split(r.Form.Get("scope"), " "),
		})
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
	//scope := middlewares.GetScope(r.Context())
	//currentSessionService := ioc.Get[services.CurrentSessionService](scope)
	//
	//routeParams := mux.Vars(r)
	//realmName := routeParams["realmName"]
	//currentSessionService.DeleteSession(w, realmName)

	// TODO: the oidc logout mechanism is distinct from the "normal" logout
	//		 oidc logout should only log the user out of the client that requested the logout
	//		 normal logout should log the user out of the holvit session, but not (usually) invalidate the tokens of clients that were signed in

	// TODO: when a user signs into a client and has already authenticated and authorized previously,
	// 		 instead of redirecting them immediately they should be prompted if they want to sign into that client with that account
	//		 this choice should be remembered for the browser session (or longer) or until the user logs out of that client via the oidc logout

}

func Discovery(w http.ResponseWriter, r *http.Request) {
}

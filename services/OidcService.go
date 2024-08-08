package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"holvit/cache"
	"holvit/constants"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
	"holvit/routes"
	"holvit/utils"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AuthorizationRequest struct {
	ResponseTypes       []string `json:"responseTypes"`
	RealmName           string   `json:"realmName"`
	ClientId            string   `json:"clientId"`
	RedirectUri         string   `json:"redirectUri"`
	Scopes              []string `json:"scopes"`
	State               string   `json:"state"`
	ResponseMode        string   `json:"responseMode"`
	PKCEChallenge       string   `json:"pkceChallenge"`
	PKCEChallengeMethod string   `json:"pkceChallengeMethod"`
}

type AuthorizationResponse interface {
	HandleHttp(w http.ResponseWriter, r *http.Request)
}

type ScopeConsentResponse struct {
	RequiredGrants []repos.Scope
	Client         *repos.Client
	User           *repos.User
	Token          string
	RedirectUri    string
}

func (c *ScopeConsentResponse) HandleHttp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	scopes := make([]AuthFrontendScope, 0, len(c.RequiredGrants))
	for _, grant := range c.RequiredGrants {
		scopes = append(scopes, AuthFrontendScope{
			Required:    grant.Name == "openid", // TODO: idk what lol
			Name:        grant.Name,
			DisplayName: grant.DisplayName,
			Description: grant.Description,
		})
	}

	realmName := "admin" // TODO: get from where?

	frontendData := AuthFrontendData{
		Mode: constants.FrontendModeAuthorize,
		Authorize: &AuthFrontendDataAuthorize{
			ClientName: c.Client.DisplayName,
			User: AuthFrontendUser{
				Name: c.User.Username,
			},
			Scopes:    scopes,
			Token:     c.Token,
			RefuseUrl: c.RedirectUri, // TODO: are we sure?
			LogoutUrl: routes.OidcLogout.Url(realmName),
			GrantUrl:  routes.AuthorizeGrant.Url(realmName),
		},
	}

	frontendService := ioc.Get[FrontendService](scope)

	err := frontendService.WriteAuthFrontend(w, realmName, frontendData)
	if err != nil {
		rcs.Error(err)
	}
}

type CodeAuthorizationResponse struct {
	Code        string
	RedirectUri string
	State       string
}

func (c *CodeAuthorizationResponse) BuildRedirectUri() (string, error) {
	redirectUri, err := url.Parse(c.RedirectUri)
	if err != nil {
		return "", err
	}

	query := redirectUri.Query()
	query.Add("code", c.Code)

	if c.State != "" {
		query.Add("state", c.State)
	}

	redirectUri.RawQuery = query.Encode()
	return redirectUri.String(), nil
}

func (c *CodeAuthorizationResponse) HandleHttp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	uri, err := c.BuildRedirectUri()
	if err != nil {
		rcs.Error(err)
		return
	}
	http.Redirect(w, r, uri, http.StatusFound)
}

type GrantRequest struct {
	ClientId             uuid.UUID
	RealmId              uuid.UUID
	ScopeNames           []string
	AuthorizationRequest AuthorizationRequest
}

type AuthorizationCodeTokenRequest struct {
	RedirectUri  string
	Code         string
	ClientId     string
	ClientSecret h.Optional[string]
	PKCEVerifier h.Optional[string]
}

type RefreshTokenRequest struct {
	RefreshToken string
	ClientId     string
	ClientSecret h.Optional[string]
	ScopeNames   []string
}

type TokenResponse struct {
	TokenType string `json:"token_type"`

	IdToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	Scope *string `json:"scope"`

	ExpiresIn int `json:"expires_in"`
}

type OidcService interface {
	Authorize(ctx context.Context, authorizationRequest AuthorizationRequest) (AuthorizationResponse, error)
	Grant(ctx context.Context, grantRequest GrantRequest) (AuthorizationResponse, error)
	HandleAuthorizationCode(ctx context.Context, request AuthorizationCodeTokenRequest) (*TokenResponse, error)
	HandleRefreshToken(ctx context.Context, request RefreshTokenRequest) (*TokenResponse, error)
	UserInfo(bearer string) map[string]interface{}
}

type OidcServiceImpl struct{}

func NewOidcService() OidcService {
	return &OidcServiceImpl{}
}

func (o *OidcServiceImpl) HandleAuthorizationCode(ctx context.Context, request AuthorizationCodeTokenRequest) (*TokenResponse, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	tokenService := ioc.Get[TokenService](scope)
	codeInfo, err := tokenService.RetrieveOidcCode(ctx, request.Code)
	if err != nil {
		return nil, err
	}

	if request.RedirectUri != codeInfo.RedirectUri {
		return nil, httpErrors.Unauthorized().WithMessage("invalid redirect uri")
	}

	clientService := ioc.Get[ClientService](scope)
	client, err := clientService.Authenticate(ctx, AuthenticateClientRequest{
		ClientId:     request.ClientId,
		ClientSecret: request.ClientSecret,
	})
	if err != nil {
		return nil, err
	}
	if codeInfo.ClientId != client.ClientId {
		return nil, httpErrors.Unauthorized().WithMessage("wrong client id")
	}

	if client.ClientSecret.IsNone() {
		codeChallenge := codeInfo.PKCEChallenge.UnwrapErr(httpErrors.Unauthorized().WithMessage("PKCE required"))
		codeVerifier := request.PKCEVerifier.UnwrapErr(httpErrors.Unauthorized().WithMessage("PKCE required"))
		hashedVerifier := base64.URLEncoding.EncodeToString(utils.Sha256(codeVerifier))

		if codeChallenge != hashedVerifier { // don't need constant time compare because we're comparing hashes
			return nil, httpErrors.Unauthorized().WithMessage("wrong PKCE code verifier")
		}
	}

	claimsService := ioc.Get[ClaimsService](scope)
	claims, err := claimsService.GetClaims(ctx, GetClaimsRequest{
		UserId:   codeInfo.UserId,
		ScopeIds: codeInfo.GrantedScopeIds,
	})
	if err != nil {
		return nil, err
	}

	idTokenValidTime := time.Hour * 1     //TODO: add this to realm and maybe to scopes
	accessTokenValidTime := time.Hour * 1 //TODO: add this to realm and maybe to scopes

	issuer := "http://localhost:8080/oidc" //TODO: this needs to be in the config (external url)
	audience := client.ClientId

	idTokenClaims := jwt.MapClaims{
		"sub": codeInfo.UserId.String(),
		"iss": issuer,
		"aud": audience,
		"iat": now.Unix(),
		"exp": now.Add(idTokenValidTime).Unix(),
	}

	for _, claim := range claims {
		idTokenClaims[claim.Name] = claim.Claim
	}

	subject := idTokenClaims["sub"].(string)

	idToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, idTokenClaims)

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub":    codeInfo.UserId,
		"scopes": codeInfo.GrantedScopes,
		"iat":    now.Unix(),
		"exp":    now.Add(accessTokenValidTime).Unix(),
	})

	keyCache := ioc.Get[cache.KeyCache](scope)
	key, ok := keyCache.Get(client.RealmId)
	if !ok {
		return nil, httpErrors.Unauthorized().WithMessage("could not get realm key")
	}

	idTokenString, err := idToken.SignedString(key)
	if err != nil {
		return nil, err
	}

	accessTokenString, err := accessToken.SignedString(key)
	if err != nil {
		return nil, err
	}

	refreshTokenService := ioc.Get[RefreshTokenService](scope)
	refreshTokenString, _, err := refreshTokenService.CreateRefreshToken(ctx, CreateRefreshTokenRequest{
		ClientId: client.Id,
		UserId:   codeInfo.UserId,
		RealmId:  client.RealmId,
		Issuer:   issuer,
		Subject:  subject,
		Audience: audience,
		Scopes:   codeInfo.GrantedScopes,
	})
	if err != nil {
		return nil, err
	}

	scopeString := strings.Join(codeInfo.GrantedScopes, " ")
	return &TokenResponse{
		TokenType:    "Bearer",
		IdToken:      idTokenString,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		Scope:        &scopeString,
		ExpiresIn:    int(accessTokenValidTime / time.Second),
	}, nil
}

func (o *OidcServiceImpl) HandleRefreshToken(ctx context.Context, request RefreshTokenRequest) (*TokenResponse, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	clientService := ioc.Get[ClientService](scope)
	client, err := clientService.Authenticate(ctx, AuthenticateClientRequest{
		ClientId:     request.ClientId,
		ClientSecret: request.ClientSecret,
	})
	if err != nil {
		return nil, err
	}

	refreshTokenService := ioc.Get[RefreshTokenService](scope)
	refreshTokenString, refreshToken, err := refreshTokenService.ValidateAndRefresh(ctx, request.RefreshToken, client.Id)
	if err != nil {
		return nil, err
	}

	if !utils.IsSliceSubset(refreshToken.Scopes, request.ScopeNames) {
		return nil, httpErrors.Unauthorized().WithMessage("too many scopes")
	}

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	scopes := scopeRepository.FindScopes(ctx, repos.ScopeFilter{
		RealmId: refreshToken.RealmId,
		Names:   h.Some(request.ScopeNames),
	})

	grantedScopeIds := make([]uuid.UUID, 0)
	for _, dbScope := range scopes.Values() {
		grantedScopeIds = append(grantedScopeIds, dbScope.Id)
	}

	claimsService := ioc.Get[ClaimsService](scope)
	claims, err := claimsService.GetClaims(ctx, GetClaimsRequest{
		UserId:   refreshToken.UserId,
		ScopeIds: grantedScopeIds,
	})
	if err != nil {
		return nil, err
	}

	accessTokenValidTime := time.Hour * 1 //TODO: add this to realm and maybe to scopes
	idTokenValidTime := time.Hour * 1     //TODO: add this to realm and maybe to scopes

	idTokenClaims := jwt.MapClaims{
		"sub": refreshToken.Subject,
		"iss": refreshToken.Issuer,
		"aud": refreshToken.Audience,
		"iat": now.Unix(),
		"exp": now.Add(idTokenValidTime).Unix(),
	}

	for _, claim := range claims {
		idTokenClaims[claim.Name] = claim.Claim
	}

	idToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, idTokenClaims)

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub":    refreshToken.UserId,
		"scopes": request.ScopeNames,
		"iat":    now.Unix(),
		"exp":    now.Add(accessTokenValidTime).Unix(),
	})

	keyCache := ioc.Get[cache.KeyCache](scope)
	key, ok := keyCache.Get(client.RealmId)
	if !ok {
		return nil, httpErrors.Unauthorized().WithMessage("could not get realm key")
	}

	idTokenString, err := idToken.SignedString(key)
	if err != nil {
		return nil, err
	}

	accessTokenString, err := accessToken.SignedString(key)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		TokenType:    "Bearer",
		IdToken:      idTokenString,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int(accessTokenValidTime / time.Second),
	}, nil
}

func (o *OidcServiceImpl) Grant(ctx context.Context, grantRequest GrantRequest) (AuthorizationResponse, error) {
	scope := middlewares.GetScope(ctx)

	currentUserService := ioc.Get[CurrentSessionService](scope)

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	scopes := scopeRepository.FindScopes(ctx, repos.ScopeFilter{
		RealmId: grantRequest.RealmId,
		Names:   h.Some(grantRequest.ScopeNames),
	})

	scopeIds := make([]uuid.UUID, 0)
	for _, scope := range scopes.Values() {
		scopeIds = append(scopeIds, scope.Id)
	}

	userId, err := currentUserService.UserId()
	if err != nil {
		return nil, err
	}

	scopeRepository.CreateGrants(ctx, userId, grantRequest.ClientId, scopeIds)

	return o.Authorize(ctx, grantRequest.AuthorizationRequest)
}

func (o *OidcServiceImpl) Authorize(ctx context.Context, authorizationRequest AuthorizationRequest) (AuthorizationResponse, error) {
	if !(len(authorizationRequest.ResponseTypes) == 1 && authorizationRequest.ResponseTypes[0] == constants.AuthorizationResponseTypeCode) {
		return nil, httpErrors.BadRequest().WithMessage("Unsupported authorization flow, only supporting 'code'")
	}

	if authorizationRequest.ResponseMode == "" {
		authorizationRequest.ResponseMode = constants.AuthorizationResponseModeQuery
	}
	err := validateResponseMode(authorizationRequest.ResponseMode)
	if err != nil {
		return nil, err
	}

	scope := middlewares.GetScope(ctx)

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealms(ctx, repos.RealmFilter{
		Name: h.Some(authorizationRequest.RealmName),
	}).First()

	clientRepository := ioc.Get[repos.ClientRepository](scope)
	client := clientRepository.FindClients(ctx, repos.ClientFilter{
		RealmId:  h.Some(realm.Id),
		ClientId: h.Some(authorizationRequest.ClientId),
	}).First()

	pkceChallenge := h.None[string]()
	if client.ClientSecret.IsSome() && authorizationRequest.PKCEChallenge != "" {
		return nil, httpErrors.BadRequest().WithMessage("clients with a secret cannot use PKCE")
	} else if client.ClientSecret.IsNone() {
		if authorizationRequest.PKCEChallenge == "" {
			return nil, httpErrors.BadRequest().WithMessage("clients without a secret must use PKCE")
		}
		if authorizationRequest.PKCEChallengeMethod != constants.CodeChallengeMethodS256 {
			return nil, httpErrors.BadRequest().WithMessage(fmt.Sprintf("Unsupported PKCE code challenge method '%v'", authorizationRequest.PKCEChallengeMethod))
		}
		pkceChallenge = h.Some(authorizationRequest.PKCEChallenge)
	}

	currentUser := ioc.Get[CurrentSessionService](scope)

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	userid, err := currentUser.UserId()
	if err != nil {
		return nil, err
	}

	scopes := scopeRepository.FindScopes(ctx, repos.ScopeFilter{
		Names:         h.Some(authorizationRequest.Scopes),
		UserId:        h.Some(userid),
		ClientId:      h.Some(client.Id),
		RealmId:       realm.Id,
		IncludeGrants: true,
	})

	missingGrants := make([]repos.Scope, 0)
	for _, oidcScope := range scopes.Values() {
		if oidcScope.Grant == nil {
			missingGrants = append(missingGrants, oidcScope)
		}
	}

	//TODO: only on first round
	if len(missingGrants) > 0 {
		tokenService := ioc.Get[TokenService](scope)
		token := tokenService.StoreGrantInfo(ctx, GrantInfo{
			RealmId:              realm.Id,
			ClientId:             client.Id,
			AuthorizationRequest: authorizationRequest,
		})

		user := currentUser.User(ctx)

		return &ScopeConsentResponse{
			RequiredGrants: missingGrants,
			Token:          token.Unwrap(),
			Client:         &client,
			User:           user.Unwrap(),
			RedirectUri:    authorizationRequest.RedirectUri,
		}, nil
	}

	grantedScopes := make([]string, 0)
	grantedScopeIds := make([]uuid.UUID, 0)
	for _, oidcScope := range scopes.Values() {
		if oidcScope.Grant != nil {
			grantedScopes = append(grantedScopes, oidcScope.Name)
			grantedScopeIds = append(grantedScopeIds, oidcScope.Id)
		}
	}

	tokenService := ioc.Get[TokenService](scope)
	code, err := tokenService.StoreOidcCode(ctx, CodeInfo{
		RealmId:       realm.Id,
		ClientId:      client.ClientId,
		UserId:        userid,
		RedirectUri:   authorizationRequest.RedirectUri,
		GrantedScopes: grantedScopes,
		PKCEChallenge: pkceChallenge,
	})
	if err != nil {
		return nil, err
	}

	return &CodeAuthorizationResponse{
		Code:        code,
		RedirectUri: authorizationRequest.RedirectUri,
		State:       authorizationRequest.State,
	}, nil
}

func validateResponseMode(responseMode string) error {
	if responseMode == constants.AuthorizationResponseModeQuery {
		return nil
	}
	if responseMode == "" {
		return nil
	}
	return httpErrors.BadRequest().WithMessage(fmt.Sprintf("Unsupported response mode %v", responseMode))
}

func (o *OidcServiceImpl) UserInfo(bearer string) map[string]interface{} {

	return nil
}

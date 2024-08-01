package services

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"holvit/cache"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/requestContext"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AuthorizationRequest struct {
	ResponseTypes []string
	RealmName     string
	ClientId      string
	RedirectUri   string
	Scopes        []string
	State         string
	ResponseMode  string
}

type AuthorizationResponse interface {
	HandleHttp(w http.ResponseWriter, r *http.Request)
}

type ScopeConsentResponse struct {
	RequiredGrants []string
	Token          string
}

func (c *ScopeConsentResponse) HandleHttp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	list := "<ul>"
	fields := ""
	for _, missingGrant := range c.RequiredGrants {
		list += "<li>" + missingGrant + "</li>"
		fields += "<input type='hidden' name='grant' value='" + missingGrant + "'/>"
	}
	list += "</ul>"

	_, err := w.Write([]byte("<html><h1>Consent Required!</h1>" + list +
		"<form action='/oidc/authorize-grant' method='post'><input type='hidden' name='token' value='" + c.Token + "'/>" +
		fields + "<input type='submit' value='gib'/></form></html>"))
	if err != nil {
		rcs.Error(err)
		return
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
	ClientSecret string
}

type RefreshTokenRequest struct {
	RefreshToken string
	ClientId     string
	ClientSecret string
}

type TokenResponse struct {
	TokenType string `json:"token_type"`

	IdToken      *string `json:"id_token"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`

	Scope string `json:"scope"`

	ExpiresIn int `json:"expires_in"`
}

type OidcService interface {
	Authorize(ctx context.Context, authorizationRequest AuthorizationRequest) (AuthorizationResponse, error)
	Grant(ctx context.Context, grantRequest GrantRequest) (AuthorizationResponse, error)
	HandleAuthorizationCode(ctx context.Context, request AuthorizationCodeTokenRequest) (*TokenResponse, error)
	HandleRefreshToken(ctx context.Context, request RefreshTokenRequest) (*TokenResponse, error)
}

type OidcServiceImpl struct{}

func NewOidcService() OidcService {
	return &OidcServiceImpl{}
}

func (o *OidcServiceImpl) HandleAuthorizationCode(ctx context.Context, request AuthorizationCodeTokenRequest) (*TokenResponse, error) {
	scope := middlewares.GetScope(ctx)

	tokenService := ioc.Get[TokenService](scope)
	codeInfo, err := tokenService.RetrieveOidcCode(ctx, request.Code)
	if err != nil {
		return nil, err
	}

	if request.RedirectUri != codeInfo.RedirectUri {
		return nil, httpErrors.Unauthorized()
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
		return nil, httpErrors.Unauthorized()
	}

	// generate jwt

	refreshTokenService := ioc.Get[RefreshTokenService](scope)
	refreshToken, err := refreshTokenService.CreateRefreshToken(ctx, CreateRefreshTokenRequest{
		ClientId: client.Id,
		UserId:   codeInfo.UserId,
		RealmId:  client.RealmId,
	})
	if err != nil {
		return nil, err
	}

	// access token
	// scopes

	accessTokenValidTime := time.Hour * 1 //TODO: add this to realm and maybe to scopes

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub":    codeInfo.UserId,
		"scopes": codeInfo.GrantedScopes,
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(accessTokenValidTime).Unix(),
	})

	keyCache := ioc.Get[cache.KeyCache](scope)
	// Sign and get the complete encoded token as a string using the secret
	key, ok := keyCache.Get(client.RealmId)
	if !ok {
		return nil, httpErrors.Unauthorized()
	}
	tokenString, err := token.SignedString(key)

	return &TokenResponse{
		TokenType:    "Bearer",
		IdToken:      nil,
		AccessToken:  tokenString,
		RefreshToken: refreshToken,
		Scope:        strings.Join(codeInfo.GrantedScopes, " "),
		ExpiresIn:    int(accessTokenValidTime / time.Second),
	}, nil
}

func (o *OidcServiceImpl) HandleRefreshToken(ctx context.Context, request RefreshTokenRequest) (*TokenResponse, error) {
	return &TokenResponse{}, nil
}

func (o *OidcServiceImpl) Grant(ctx context.Context, grantRequest GrantRequest) (AuthorizationResponse, error) {
	scope := middlewares.GetScope(ctx)

	// assume we are already logged in
	userRepository := ioc.Get[repositories.UserRepository](scope)
	users, _, err := userRepository.FindUsers(ctx, repositories.UserFilter{
		BaseFilter: repositories.BaseFilter{},
	})
	if err != nil {
		return nil, err
	}
	adminUser := users[0]

	scopeRepository := ioc.Get[repositories.ScopeRepository](scope)
	scopes, _, err := scopeRepository.FindScopes(ctx, repositories.ScopeFilter{
		RealmId: grantRequest.RealmId,
		Names:   grantRequest.ScopeNames,
	})
	if err != nil {
		return nil, err
	}

	scopeIds := make([]uuid.UUID, 0, len(scopes))
	for _, scope := range scopes {
		scopeIds = append(scopeIds, scope.Id)
	}

	err = scopeRepository.CreateGrants(ctx, adminUser.Id, grantRequest.ClientId, scopeIds)
	if err != nil {
		return nil, err
	}

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

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realms, count, err := realmRepository.FindRealms(ctx, repositories.RealmFilter{
		Name: &authorizationRequest.RealmName,
	})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, httpErrors.NotFound().WithMessage("Realm not found")
	}
	realm := realms[0]

	clientRepository := ioc.Get[repositories.ClientRepository](scope)
	clients, count, err := clientRepository.FindClients(ctx, repositories.ClientFilter{
		RealmId:  &realm.Id,
		ClientId: &authorizationRequest.ClientId,
	})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, httpErrors.NotFound().WithMessage("Client not found")
	}
	client := clients[0]

	// assume we are already logged in
	userRepository := ioc.Get[repositories.UserRepository](scope)
	users, _, err := userRepository.FindUsers(ctx, repositories.UserFilter{
		BaseFilter: repositories.BaseFilter{},
	})
	if err != nil {
		return nil, err
	}
	adminUser := users[0]

	scopeRepository := ioc.Get[repositories.ScopeRepository](scope)
	scopes, count, err := scopeRepository.FindScopes(ctx, repositories.ScopeFilter{
		Names:         authorizationRequest.Scopes,
		UserId:        &adminUser.Id,
		ClientId:      &client.Id,
		RealmId:       realm.Id,
		IncludeGrants: true,
	})
	if err != nil {
		return nil, err
	}

	missingGrants := make([]string, 0, len(scopes))
	for _, oidcScope := range scopes {
		if oidcScope.Grant == nil {
			missingGrants = append(missingGrants, oidcScope.Name)
		}
	}

	//TODO: only on first round!
	if len(missingGrants) > 0 {
		tokenService := ioc.Get[TokenService](scope)
		token, err := tokenService.StoreGrantInfo(ctx, GrantInfo{
			RealmId:              realm.Id,
			ClientId:             client.Id,
			AuthorizationRequest: authorizationRequest,
		})
		if err != nil {
			return nil, err
		}
		return &ScopeConsentResponse{
			RequiredGrants: missingGrants,
			Token:          token,
		}, nil
	}

	grantedScopes := make([]string, 0, len(scopes))
	for _, oidcScope := range scopes {
		if oidcScope.Grant != nil {
			grantedScopes = append(grantedScopes, oidcScope.Name)
		}
	}

	tokenService := ioc.Get[TokenService](scope)
	code, err := tokenService.StoreOidcCode(ctx, CodeInfo{
		RealmId:       realm.Id,
		ClientId:      client.ClientId,
		UserId:        adminUser.Id,
		RedirectUri:   authorizationRequest.RedirectUri,
		GrantedScopes: grantedScopes,
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
	//TODO: add these when we support more response modes
	/*if responseMode == constants.AuthorizationResponseModeFragment {
		return nil
	}
	if responseMode == constants.AuthorizationResponseModeFormPost {
		return nil
	}*/
	if responseMode == "" {
		return nil
	}
	return httpErrors.BadRequest().WithMessage(fmt.Sprintf("Unsupported response mode %v", responseMode))
}

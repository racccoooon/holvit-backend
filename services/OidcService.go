package services

import (
	"context"
	"fmt"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/requestContext"
	"net/http"
	"net/url"
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
}

func (c *ScopeConsentResponse) HandleHttp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	list := "<ul>"
	for _, missingGrant := range c.RequiredGrants {
		list += "<li>" + missingGrant + "</li>"
	}
	list += "</ul>"

	_, err := w.Write([]byte("<html><h1>Consent Required!</h1>" + list + "</html>"))
	if err != nil {
		rcs.Error(err)
		return
	}
}

type CodeAuthorizationResponse struct {
	request AuthorizationRequest
	code    string
}

func (c *CodeAuthorizationResponse) BuildRedirectUri() (string, error) {
	redirectUri, err := url.Parse(c.request.RedirectUri)
	if err != nil {
		return "", err
	}

	query := redirectUri.Query()
	query.Add("code", c.code)

	if c.request.State != "" {
		query.Add("state", c.request.State)
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

type OidcService interface {
	Authorize(ctx context.Context, authorizationRequest AuthorizationRequest) (AuthorizationResponse, error)
}

type OidcServiceImpl struct{}

func NewOidcService() OidcService {
	return &OidcServiceImpl{}
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

	if len(missingGrants) > 0 {
		return &ScopeConsentResponse{
			RequiredGrants: missingGrants,
		}, nil
	}

	return &CodeAuthorizationResponse{
		request: authorizationRequest,
		code:    "foobar",
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

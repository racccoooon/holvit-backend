package auth

import (
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/services"
	"net/http"
)

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
	info := tokenService.RetrieveGrantInfo(ctx, token).UnwrapErr(httpErrors.BadRequest().WithMessage("token not found"))

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

package api

import (
	"github.com/google/uuid"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"net/http"
)

type ScopeResponse struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
}

func mapScopeResponse(scope repos.Scope) ScopeResponse {
	return ScopeResponse{
		Id:          scope.Id,
		Name:        scope.Name,
		DisplayName: scope.DisplayName,
		Description: scope.Description,
	}
}

func FindScopes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	realm := getRequestRealm(r)

	filter := repos.ScopeFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: pagingFromQuery(r),
		},
		RealmId: realm.Id,
	}

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	scopes := scopeRepository.FindScopes(ctx, filter)

	rows := make([]ScopeResponse, len(scopes.Values()))
	for _, scope := range scopes.Values() {
		rows = append(rows, mapScopeResponse(scope))
	}

	writeFindResponse(w, rows, scopes.Count())
}

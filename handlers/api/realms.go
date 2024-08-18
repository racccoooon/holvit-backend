package api

import (
	"github.com/google/uuid"
	"github.com/sourcegraph/conc/iter"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"net/http"
)

type RealmResponse struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
}

func mapRealmResponse(realm *repos.Realm) RealmResponse {
	return RealmResponse{
		Id:          realm.Id,
		Name:        realm.Name,
		DisplayName: realm.DisplayName,
	}
}

func FindRealms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realms := realmRepository.FindRealms(ctx, repos.RealmFilter{})

	rows := iter.Map(realms.Values(), mapRealmResponse)

	writeFindResponse(w, rows, realms.Count())
}

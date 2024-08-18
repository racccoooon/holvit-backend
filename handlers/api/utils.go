package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"net/http"
	"strconv"
)

type FindResponse[T any] struct {
	TotalCount int `json:"totalCount"`
	Rows       []T `json:"rows"`
}

type QuerySortOrder struct {
	SortField string
	Ascending bool
}

func (q QuerySortOrder) MapAllowed(allowed map[string]string) repos.SortInfo {
	if mapped, ok := allowed[q.SortField]; ok {
		return repos.SortInfo{
			Field:     mapped,
			Ascending: false,
		}
	}
	panic(httpErrors.BadRequest().WithMessage(fmt.Sprintf("unsupported sort field '%s'", q.SortField)))
}

func searchTextFromQuery(r *http.Request) h.Opt[string] {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	searchText := r.Form.Get("q")
	if searchText == "" {
		return h.None[string]()
	}
	return h.Some(searchText)
}

func sortFromQuery(r *http.Request) h.Opt[QuerySortOrder] {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	sortDirection := r.Form.Get("dir")
	if sortDirection == "" {
		sortDirection = "asc"
	}

	if sortDirection != "asc" && sortDirection != "desc" {
		panic(httpErrors.BadRequest().WithMessage("invalid query parameter 'dir', must be 'asc' or 'desc'"))
	}

	sortField := r.Form.Get("sort")
	if sortField == "" {
		return h.None[QuerySortOrder]()
	}

	ascending := sortDirection == "asc"

	return h.Some(QuerySortOrder{
		SortField: sortField,
		Ascending: ascending,
	})
}

func pagingFromQuery(r *http.Request) h.Opt[repos.PagingInfo] {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	pageSizeString := r.Form.Get("pageSize")
	if pageSizeString == "" {
		panic(httpErrors.BadRequest().WithMessage("page size parameter is required"))
	}
	pageSize, err := strconv.Atoi(pageSizeString)
	if err != nil {
		panic(httpErrors.BadRequest().WithMessage("invalid query parameter 'pageSize', must be an int"))
	}

	pageString := r.Form.Get("page")
	if pageString == "" {
		pageString = "1"
	}
	page, err := strconv.Atoi(pageString)
	if err != nil {
		panic(httpErrors.BadRequest().WithMessage("invalid query parameter 'page', must be an int"))
	}

	return h.Some(repos.PagingInfo{
		PageSize:   pageSize,
		PageNumber: page,
	})
}

func getRequestRealm(r *http.Request) repos.Realm {
	routeParams := mux.Vars(r)
	realmName := routeParams["realmName"]

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealms(ctx, repos.RealmFilter{
		Name: h.Some(realmName),
	}).Single()

	return realm
}

func writeFindResponse[T any](w http.ResponseWriter, rows []T, totalCount int) {
	response := FindResponse[T]{
		TotalCount: totalCount,
		Rows:       rows,
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		panic(err)
	}
}

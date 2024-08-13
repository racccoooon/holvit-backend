package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/services"
	"net/http"
	"strconv"
)

func pagingFromQuery(r *http.Request) h.Opt[repos.PagingInfo] {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	pageSizeString := r.Form.Get("page")
	if pageSizeString == "" {
		panic(httpErrors.BadRequest().WithMessage("page size parameter is required"))
	}
	pageSize, err := strconv.Atoi(pageSizeString)
	if err != nil {
		panic(httpErrors.BadRequest().WithMessage("invalid query parameter page size, must be an int"))
	}

	pageString := r.Form.Get("page")
	if pageString == "" {
		pageString = "1"
	}
	page, err := strconv.Atoi(pageString)
	if err != nil {
		panic(httpErrors.BadRequest().WithMessage("invalid query parameter page, must be an int"))
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
	}).First()

	return realm
}

type ApiFindResponse[T any] struct {
	TotalCount int `json:"totalCount"`
	Rows       []T `json:"rows"`
}

type ApiUserReponse struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	Email         *string   `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
}

func MapUserResponse(user repos.User) ApiUserReponse {
	return ApiUserReponse{
		Id:            user.Id,
		Username:      user.Username,
		Email:         user.Email.ToNillablePtr(),
		EmailVerified: user.EmailVerified,
	}
}

func FindUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	currentSessionService := ioc.Get[services.CurrentSessionService](scope)
	currentSessionService.VerifyAuthorized()

	realm := getRequestRealm(r)

	filter := repos.UserFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: pagingFromQuery(r),
		},
		RealmId: h.Some(realm.Id),
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	users := userRepository.FindUsers(ctx, filter)

	rows := make([]ApiUserReponse, 0)
	for _, user := range users.Values() {
		rows = append(rows, MapUserResponse(user))
	}

	response := ApiFindResponse[ApiUserReponse]{
		TotalCount: users.Count(),
		Rows:       rows,
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		panic(err)
	}
}

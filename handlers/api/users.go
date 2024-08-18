package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sourcegraph/conc/iter"
	"holvit/h"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/services"
	"net/http"
)

type CreateUserRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	request := CreateUserRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		panic(err)
	}

	realm := getRequestRealm(r)

	userService := ioc.Get[services.UserService](scope)
	userId := userService.CreateUser(ctx, services.CreateUserRequest{
		RealmId:  realm.Id,
		Username: request.Username,
		Email:    h.FromPtr(request.Email),
	}).Unwrap() // TODO: handle

	writeCreateResponse(w, userId)
}

type UserRepsonse struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	Email         *string   `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
}

func mapUserResponse(user *repos.User) UserRepsonse {
	return UserRepsonse{
		Id:            user.Id,
		Username:      user.Username,
		Email:         user.Email.ToNillablePtr(),
		EmailVerified: user.EmailVerified,
	}
}

func FindUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	realm := getRequestRealm(r)

	filter := repos.UserFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: pagingFromQuery(r),
			SortInfo: h.MapOpt(sortFromQuery(r), func(order QuerySortOrder) repos.SortInfo {
				return order.MapAllowed(map[string]string{
					"username": "username",
					"email":    "email",
				})
			}),
			SearchText: searchTextFromQuery(r),
		},
		RealmId: h.Some(realm.Id),
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	users := userRepository.FindUsers(ctx, filter)

	rows := iter.Map(users.Values(), mapUserResponse)

	writeFindResponse(w, rows, users.Count())
}

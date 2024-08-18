package api

import (
	"github.com/google/uuid"
	"github.com/sourcegraph/conc/iter"
	"holvit/h"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"net/http"
)

type UserReponse struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	Email         *string   `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
}

func mapUserResponse(user *repos.User) UserReponse {
	return UserReponse{
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

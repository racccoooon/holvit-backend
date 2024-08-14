package api

import (
	"github.com/google/uuid"
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

func mapUserResponse(user repos.User) UserReponse {
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
		},
		RealmId: h.Some(realm.Id),
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	users := userRepository.FindUsers(ctx, filter)

	rows := make([]UserReponse, len(users.Values()))
	for _, user := range users.Values() {
		rows = append(rows, mapUserResponse(user))
	}

	writeFindResponse(w, rows, users.Count())
}

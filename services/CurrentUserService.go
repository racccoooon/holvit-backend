package services

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"holvit/ioc"
	"holvit/middlewares"
	"net/http"
)

type CurrentUserService interface {
	GetCurrentUser() *CurrentUser
}

func NewCurrentUserService() CurrentUserService {
	return &CurrentUserServiceImpl{}
}

type CurrentUser struct {
	UserId  uuid.UUID
	RealmId uuid.UUID
}

type CurrentUserServiceImpl struct {
	CurrentUser *CurrentUser
}

func (s *CurrentUserServiceImpl) GetCurrentUser() *CurrentUser {
	return s.CurrentUser
}

func CurrentUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		scope := middlewares.GetScope(ctx)

		service := ioc.Get[CurrentUserService](scope)
		serviceImpl := service.(*CurrentUserServiceImpl)

		routeParams := mux.Vars(r)
		realmName := routeParams["realmName"]

		sessionToken, err := r.Cookie(fmt.Sprintf("holvit_%s_session", realmName))
		if err == nil {
			sessionService := ioc.Get[SessionService](scope)
			session, err := sessionService.ValidateSession(ctx, sessionToken.Value)
			if err == nil {
				serviceImpl.CurrentUser = &CurrentUser{
					UserId:  session.UserId,
					RealmId: session.RealmId,
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

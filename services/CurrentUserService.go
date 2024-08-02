package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"net/http"
)

type CurrentUserService interface {
	GetCurrentUser() *CurrentUser
}

func NewCurrentUserService() CurrentUserService {
	return &CurrentUserServiceImpl{}
}

type CurrentUser struct {
	UserId uuid.UUID
	user   *repositories.User

	RealmId uuid.UUID
	realm   *repositories.Realm
}

func (s *CurrentUser) GetUser(ctx context.Context) (*repositories.User, error) {
	if s.user == nil {
		scope := middlewares.GetScope(ctx)
		userRepository := ioc.Get[repositories.UserRepository](scope)
		user, err := userRepository.FindUserById(ctx, s.UserId)
		if err != nil {
			return nil, err
		}
		s.user = user
	}
	return s.user, nil
}

func (s *CurrentUser) GetRealm(ctx context.Context) (*repositories.Realm, error) {
	if s.realm == nil {
		scope := middlewares.GetScope(ctx)
		realmRepository := ioc.Get[repositories.RealmRepository](scope)
		realm, err := realmRepository.FindRealmById(ctx, s.RealmId)
		if err != nil {
			return nil, err
		}
		s.realm = realm
	}
	return s.realm, nil
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

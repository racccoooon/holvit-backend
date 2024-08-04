package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
	"net/http"
)

type CurrentUserService interface {
	VerifyAuthorized() error

	DeviceIdString() (string, error)
	DeviceId(ctx context.Context) (uuid.UUID, error)
	Device(ctx context.Context) (*repositories.UserDevice, error)

	UserId() (uuid.UUID, error)
	User(ctx context.Context) (*repositories.User, error)

	RealmId() (uuid.UUID, error)
	Realm(ctx context.Context) (*repositories.Realm, error)
}

func NewCurrentUserService() CurrentUserService {
	return &CurrentUserServiceImpl{}
}

type CurrentUserServiceImpl struct {
	deviceIdString *string
	deviceId       *uuid.UUID
	device         *repositories.UserDevice

	userId *uuid.UUID
	user   *repositories.User

	realmId *uuid.UUID
	realm   *repositories.Realm
}

func (s *CurrentUserServiceImpl) VerifyAuthorized() error {
	if s.userId == nil {
		return httpErrors.Unauthorized()
	}

	return nil
}

func (s *CurrentUserServiceImpl) DeviceIdString() (string, error) {
	if s.deviceIdString == nil {
		return "", httpErrors.BadRequest().WithMessage("Missing device id cookie")
	}

	return *s.deviceIdString, nil
}

func (s *CurrentUserServiceImpl) DeviceId(ctx context.Context) (uuid.UUID, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return uuid.UUID{}, err
	}

	scope := middlewares.GetScope(ctx)

	deviceIdString, err := s.DeviceIdString()
	if err != nil {
		return uuid.UUID{}, err
	}

	userDeviceRepository := ioc.Get[repositories.UserDeviceRepository](scope)
	devices, count, err := userDeviceRepository.FindUserDevices(ctx, repositories.UserDeviceFilter{
		DeviceId: &deviceIdString,
		UserId:   s.userId,
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	if count == 0 {
		return uuid.UUID{}, nil
	}

	s.device = devices[0]
	s.deviceId = &s.device.Id

	return *s.deviceId, nil
}

func (s *CurrentUserServiceImpl) Device(ctx context.Context) (*repositories.UserDevice, error) {
	_, err := s.DeviceId(ctx)
	if err != nil {
		return nil, err
	}

	return s.device, nil
}

func (s *CurrentUserServiceImpl) UserId() (uuid.UUID, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return uuid.UUID{}, err
	}

	return *s.userId, nil
}

func (s *CurrentUserServiceImpl) User(ctx context.Context) (*repositories.User, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return nil, err
	}

	if s.user == nil {
		scope := middlewares.GetScope(ctx)
		userRepository := ioc.Get[repositories.UserRepository](scope)
		user, err := userRepository.FindUserById(ctx, *s.userId)
		if err != nil {
			return nil, err
		}
		s.user = user
	}
	return s.user, nil
}

func (s *CurrentUserServiceImpl) RealmId() (uuid.UUID, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return uuid.UUID{}, err
	}

	return *s.realmId, nil
}

func (s *CurrentUserServiceImpl) Realm(ctx context.Context) (*repositories.Realm, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return nil, err
	}

	if s.realm == nil {
		scope := middlewares.GetScope(ctx)
		realmRepository := ioc.Get[repositories.RealmRepository](scope)
		realm, err := realmRepository.FindRealmById(ctx, *s.realmId)
		if err != nil {
			return nil, err
		}
		s.realm = realm
	}
	return s.realm, nil
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
				serviceImpl.realmId = &session.RealmId
				serviceImpl.userId = &session.UserId
			}
		}

		deviceId, err := r.Cookie("holvit_device_id")
		if err == nil {
			serviceImpl.deviceIdString = utils.NilIfDefault(&deviceId.Value)
		}

		next.ServeHTTP(w, r)
	})
}

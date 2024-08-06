package services

import (
	"context"
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"holvit/constants"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/utils"
	"net/http"
)

type CurrentUserService interface {
	VerifyAuthorized() error

	DeviceIdString() (string, error)
	DeviceId(ctx context.Context) (uuid.UUID, error)
	Device(ctx context.Context) (*repos.UserDevice, error)

	UserId() (uuid.UUID, error)
	User(ctx context.Context) h.Result[*repos.User]

	RealmId() (uuid.UUID, error)
	Realm(ctx context.Context) (*repos.Realm, error)
	SetSession(w http.ResponseWriter, userId uuid.UUID, rememberMe bool, realmName string, token string)
}

func NewCurrentUserService() CurrentUserService {
	return &CurrentUserServiceImpl{}
}

type CurrentUserServiceImpl struct {
	deviceIdString *string
	deviceId       *uuid.UUID
	device         *repos.UserDevice

	userId *uuid.UUID
	user   *repos.User

	realmId *uuid.UUID
	realm   *repos.Realm
}

func (s *CurrentUserServiceImpl) SetSession(w http.ResponseWriter, userId uuid.UUID, rememberMe bool, realmName string, token string) {
	cookie := http.Cookie{
		Name:     constants.SessionCookieName(realmName),
		Value:    token,
		Path:     "/",
		Domain:   "localhost", //TODO: get from settings
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if rememberMe {
		cookie.MaxAge = 99999999 //TODO: get from realm settings
	}
	http.SetCookie(w, &cookie)

	s.userId = &userId
}

func (s *CurrentUserServiceImpl) VerifyAuthorized() error {
	if s.userId == nil {
		return httpErrors.Unauthorized().WithMessage("not authorized")
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

	userDeviceRepository := ioc.Get[repos.UserDeviceRepository](scope)
	devices := userDeviceRepository.FindUserDevices(ctx, repos.UserDeviceFilter{
		DeviceId: h.Some(deviceIdString),
		UserId:   h.FromPtr(s.userId),
	})

	if devices.Count() == 0 {
		return uuid.UUID{}, nil
	}

	s.device = utils.Ptr(devices.First())
	s.deviceId = &s.device.Id

	return *s.deviceId, nil
}

func (s *CurrentUserServiceImpl) Device(ctx context.Context) (*repos.UserDevice, error) {
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

func (s *CurrentUserServiceImpl) User(ctx context.Context) h.Result[*repos.User] {
	if err := s.VerifyAuthorized(); err != nil {
		return h.Err[*repos.User](err)
	}

	if s.user == nil {
		scope := middlewares.GetScope(ctx)
		userRepository := ioc.Get[repos.UserRepository](scope)
		user := userRepository.FindUserById(ctx, *s.userId).Unwrap()
		s.user = &user
	}
	return h.Ok(s.user)
}

func (s *CurrentUserServiceImpl) RealmId() (uuid.UUID, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return uuid.UUID{}, err
	}

	return *s.realmId, nil
}

func (s *CurrentUserServiceImpl) Realm(ctx context.Context) (*repos.Realm, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return nil, err
	}

	if s.realm == nil {
		scope := middlewares.GetScope(ctx)
		realmRepository := ioc.Get[repos.RealmRepository](scope)
		realm := realmRepository.FindRealmById(ctx, *s.realmId).Unwrap()
		s.realm = &realm
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

		sessionToken, err := r.Cookie(constants.SessionCookieName(realmName))
		if err == nil {
			sessionService := ioc.Get[SessionService](scope)
			session, err := sessionService.ValidateSession(ctx, sessionToken.Value)
			if err == nil {
				serviceImpl.realmId = &session.RealmId
				serviceImpl.userId = &session.UserId
			}
		}

		deviceIdCookie, err := r.Cookie(constants.DeviceCookieName)
		if err != nil {
			deviceUuid, err := uuid.NewRandom()
			if err != nil {
				logging.Logger.Fatal(err)
			}
			deviceIdString := base64.StdEncoding.EncodeToString([]byte(deviceUuid.String()))

			http.SetCookie(w, &http.Cookie{
				Name:     constants.DeviceCookieName,
				Value:    deviceIdString,
				Path:     "/",
				Domain:   "localhost", //TODO: get from config
				MaxAge:   315360000,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			serviceImpl.deviceIdString = &deviceIdString
		} else {
			serviceImpl.deviceIdString = &deviceIdCookie.Value
		}

		next.ServeHTTP(w, r)
	})
}

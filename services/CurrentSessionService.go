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
	"time"
)

type CurrentSessionService interface {
	VerifyAuthorized() error

	DeviceIdString() (string, error)
	DeviceId(ctx context.Context) (uuid.UUID, error)
	Device(ctx context.Context) (*repos.UserDevice, error)

	UserId() (uuid.UUID, error)
	User(ctx context.Context) h.Result[*repos.User]

	RealmId() (uuid.UUID, error)
	Realm(ctx context.Context) (*repos.Realm, error)
	SetSession(w http.ResponseWriter, userId uuid.UUID, rememberMe bool, realmName string, token string)
	DeleteSession(w http.ResponseWriter, realmName string)
}

func NewCurrentSessionService() CurrentSessionService {
	return &CurrentSessionServiceImpl{}
}

type CurrentSessionServiceImpl struct {
	deviceIdString *string
	deviceId       *uuid.UUID
	device         *repos.UserDevice

	userId *uuid.UUID
	user   *repos.User

	realmId *uuid.UUID
	realm   *repos.Realm
}

func (s *CurrentSessionServiceImpl) DeleteSession(w http.ResponseWriter, realmName string) {
	setCookie(w, constants.SessionCookieName(realmName), "", -1)
}

func (s *CurrentSessionServiceImpl) SetSession(w http.ResponseWriter, userId uuid.UUID, rememberMe bool, realmName string, token string) {
	maxAge := 0
	if rememberMe {
		maxAge = int((24 * 14 * time.Hour).Seconds())
	}
	setCookie(w, constants.SessionCookieName(realmName), token, maxAge)

	s.userId = &userId
}

func (s *CurrentSessionServiceImpl) VerifyAuthorized() error {
	if s.userId == nil {
		return httpErrors.Unauthorized().WithMessage("not authorized")
	}

	return nil
}

func (s *CurrentSessionServiceImpl) DeviceIdString() (string, error) {
	if s.deviceIdString == nil {
		return "", httpErrors.BadRequest().WithMessage("Missing device id cookie")
	}

	return *s.deviceIdString, nil
}

func (s *CurrentSessionServiceImpl) DeviceId(ctx context.Context) (uuid.UUID, error) {
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

func (s *CurrentSessionServiceImpl) Device(ctx context.Context) (*repos.UserDevice, error) {
	_, err := s.DeviceId(ctx)
	if err != nil {
		return nil, err
	}

	return s.device, nil
}

func (s *CurrentSessionServiceImpl) UserId() (uuid.UUID, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return uuid.UUID{}, err
	}

	return *s.userId, nil
}

func (s *CurrentSessionServiceImpl) User(ctx context.Context) h.Result[*repos.User] {
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

func (s *CurrentSessionServiceImpl) RealmId() (uuid.UUID, error) {
	if err := s.VerifyAuthorized(); err != nil {
		return uuid.UUID{}, err
	}

	return *s.realmId, nil
}

func (s *CurrentSessionServiceImpl) Realm(ctx context.Context) (*repos.Realm, error) {
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

func setCookie(w http.ResponseWriter, name string, value string, maxAge int) {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   "localhost", //TODO: get from settings
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
}

func CurrentSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		scope := middlewares.GetScope(ctx)

		service := ioc.Get[CurrentSessionService](scope)
		serviceImpl := service.(*CurrentSessionServiceImpl)

		routeParams := mux.Vars(r)
		realmName := routeParams["realmName"]

		sessionCookie, err := r.Cookie(constants.SessionCookieName(realmName))
		if err == nil {
			sessionService := ioc.Get[SessionService](scope)
			session := sessionService.LookupSession(ctx, sessionCookie.Value)
			if session, ok := session.Get(); ok {
				serviceImpl.realmId = &session.RealmId
				serviceImpl.userId = &session.UserId
				// session cookie was good, refresh it if it has a max-age so it doesn't expire too soon
				if sessionCookie.MaxAge > 0 {
					setCookie(w, constants.SessionCookieName(realmName), sessionCookie.Value, int((14 * 24 * time.Hour).Seconds()))
				}
			}
		}

		deviceIdCookie, err := r.Cookie(constants.DeviceCookieName)
		if err != nil {
			deviceUuid, err := uuid.NewRandom()
			if err != nil {
				logging.Logger.Fatal(err)
			}
			deviceIdString := base64.StdEncoding.EncodeToString([]byte(deviceUuid.String()))

			setCookie(w, constants.DeviceCookieName, deviceIdString, int((10 * 365 * 24 * time.Hour).Seconds()))
			serviceImpl.deviceIdString = &deviceIdString
		} else {
			serviceImpl.deviceIdString = &deviceIdCookie.Value
		}

		next.ServeHTTP(w, r)
	})
}

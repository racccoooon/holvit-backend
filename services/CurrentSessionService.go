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
	IsAuthorized() bool
	VerifyAuthorized()

	DeviceIdString() string
	DeviceId(ctx context.Context) uuid.UUID
	Device(ctx context.Context) repos.UserDevice

	UserId() uuid.UUID
	User(ctx context.Context) h.Result[*repos.User]

	RealmId() uuid.UUID
	Realm(ctx context.Context) repos.Realm
	SetSession(w http.ResponseWriter, userId uuid.UUID, rememberMe bool, realmName string, token string)
	DeleteSession(w http.ResponseWriter, realmName string)
}

func NewCurrentSessionService() CurrentSessionService {
	return &currentSessionServiceImpl{}
}

type currentSessionServiceImpl struct {
	deviceIdString *string
	deviceId       *uuid.UUID
	device         *repos.UserDevice

	userId *uuid.UUID
	user   *repos.User

	realmId *uuid.UUID
	realm   *repos.Realm
}

func (s *currentSessionServiceImpl) DeleteSession(w http.ResponseWriter, realmName string) {
	setCookie(w, constants.SessionCookieName(realmName), "", -1)
}

func (s *currentSessionServiceImpl) SetSession(w http.ResponseWriter, userId uuid.UUID, rememberMe bool, realmName string, token string) {
	maxAge := 0
	if rememberMe {
		maxAge = int((24 * 14 * time.Hour).Seconds())
	}
	setCookie(w, constants.SessionCookieName(realmName), token, maxAge)

	s.userId = &userId
}

func (s *currentSessionServiceImpl) IsAuthorized() bool {
	return s.userId != nil
}

func (s *currentSessionServiceImpl) VerifyAuthorized() {
	if s.userId == nil {
		panic(httpErrors.Unauthorized().WithMessage("not authorized"))
	}
}

func (s *currentSessionServiceImpl) DeviceIdString() string {
	if s.deviceIdString == nil {
		panic(httpErrors.BadRequest().WithMessage("Missing device id cookie"))
	}

	return *s.deviceIdString
}

func (s *currentSessionServiceImpl) DeviceId(ctx context.Context) uuid.UUID {
	s.VerifyAuthorized()

	scope := middlewares.GetScope(ctx)

	deviceIdString := s.DeviceIdString()

	userDeviceRepository := ioc.Get[repos.UserDeviceRepository](scope)
	devices := userDeviceRepository.FindUserDevices(ctx, repos.UserDeviceFilter{
		DeviceId: h.Some(deviceIdString),
		UserId:   h.FromPtr(s.userId),
	})

	if devices.Count() == 0 {
		panic(httpErrors.NotFound().WithMessage("Device not found")) //TODO: maybe different error
	}

	s.device = utils.Ptr(devices.Single())
	s.deviceId = &s.device.Id

	return *s.deviceId
}

func (s *currentSessionServiceImpl) Device(ctx context.Context) repos.UserDevice {
	_ = s.DeviceId(ctx)
	return *s.device
}

func (s *currentSessionServiceImpl) UserId() uuid.UUID {
	s.VerifyAuthorized()
	return *s.userId
}

func (s *currentSessionServiceImpl) User(ctx context.Context) h.Result[*repos.User] {
	s.VerifyAuthorized()

	if s.user == nil {
		scope := middlewares.GetScope(ctx)
		userRepository := ioc.Get[repos.UserRepository](scope)
		user := userRepository.FindUserById(ctx, *s.userId).Unwrap()
		s.user = &user
	}
	return h.Ok(s.user)
}

func (s *currentSessionServiceImpl) RealmId() uuid.UUID {
	s.VerifyAuthorized()

	return *s.realmId
}

func (s *currentSessionServiceImpl) Realm(ctx context.Context) repos.Realm {
	s.VerifyAuthorized()

	if s.realm == nil {
		scope := middlewares.GetScope(ctx)
		realmRepository := ioc.Get[repos.RealmRepository](scope)
		realm := realmRepository.FindRealmById(ctx, *s.realmId).Unwrap()
		s.realm = &realm
	}

	return *s.realm
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
		serviceImpl := service.(*currentSessionServiceImpl)

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

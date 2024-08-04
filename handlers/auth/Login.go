package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/requestContext"
	"holvit/services"
	"net/http"
)

type LoginRequest struct {
	Token string `json:"token"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var request ResetPasswordRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		rcs.Error(err)
		return
	}

	tokenService := ioc.Get[services.TokenService](scope)
	loginInfo, err := tokenService.PeekLoginCode(ctx, request.Token)
	if err != nil {
		rcs.Error(err)
		return
	}

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realm, err := realmRepository.FindRealmById(ctx, loginInfo.RealmId)
	if err != nil {
		rcs.Error(err)
		return
	}

	currentUser := ioc.Get[services.CurrentUserService](scope)
	deviceIdString, err := currentUser.DeviceIdString()
	if err != nil {
		rcs.Error(err)
		return
	}
	if deviceIdString != loginInfo.DeviceId {
		rcs.Error(httpErrors.Unauthorized().WithMessage("wrong device id"))
	}

	currentStep := loginInfo.NextStep
	if currentStep != constants.AuthenticateStepSubmit {
		rcs.Error(httpErrors.Unauthorized().WithMessage(
			fmt.Sprintf("wrong login step '%s', expected '%s'", currentStep, constants.AuthenticateStepSubmit)))
		return
	}

	deviceService := ioc.Get[services.DeviceService](scope)
	isKnownDeviceResponse, err := deviceService.IsKnownUserDevice(ctx, services.IsKnownDeviceRequest{
		UserId:   loginInfo.UserId,
		DeviceId: loginInfo.DeviceId,
	})
	if err != nil {
		rcs.Error(err)
		return
	}

	deviceUuid := isKnownDeviceResponse.Id

	if !isKnownDeviceResponse.IsKnown {
		id, err := deviceService.AddKnownDevice(ctx, services.AddDeviceRequest{
			UserId:   loginInfo.UserId,
			DeviceId: deviceIdString,
		})
		if err != nil {
			rcs.Error(err)
			return
		}

		deviceUuid = id
	}

	if loginInfo.RememberMe {
		sessionService := ioc.Get[services.SessionService](scope)
		sessionToken, err := sessionService.CreateSession(ctx, services.CreateSessionRequest{
			UserId:   loginInfo.UserId,
			RealmId:  loginInfo.RealmId,
			DeviceId: *deviceUuid,
		})
		if err != nil {
			rcs.Error(err)
			return
		}
		w.Header().Add(constants.SessionCookieName(realm.Name), sessionToken)
	}

	oidcService := ioc.Get[services.OidcService](scope)
	response, err := oidcService.Authorize(ctx, loginInfo.Request)
	if err != nil {
		rcs.Error(err)
		return
	}

	response.HandleHttp(w, r)
}

type SubmitLoginStep struct {
}

func (s *SubmitLoginStep) Name() string {
	return constants.AuthenticateStepSubmit
}

func (s *SubmitLoginStep) NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error) {
	return true, nil
}

func (s *SubmitLoginStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}

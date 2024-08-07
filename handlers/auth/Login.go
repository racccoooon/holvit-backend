package auth

import (
	"context"
	"fmt"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
	"holvit/services"
	"holvit/utils"
	"net/http"
)

func CompleteAuthFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	err := r.ParseForm()
	if err != nil {
		rcs.Error(err)
		return
	}

	tokenService := ioc.Get[services.TokenService](scope)
	//TODO: i think we can get rid of all the peeks in the code and replace them with retrieve
	loginInfo, err := tokenService.RetrieveLoginCode(ctx, r.Form.Get("token"))
	if err != nil {
		rcs.Error(err)
		return
	}

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, loginInfo.RealmId).Unwrap()

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
			UserId:    loginInfo.UserId,
			DeviceId:  deviceIdString,
			UserAgent: r.UserAgent(),
			Ip:        utils.GetRequestIp(r),
		})
		if err != nil {
			rcs.Error(err)
			return
		}

		deviceUuid = id
	}

	sessionService := ioc.Get[services.SessionService](scope)
	sessionToken := sessionService.CreateSession(ctx, services.CreateSessionRequest{
		UserId:   loginInfo.UserId,
		RealmId:  loginInfo.RealmId,
		DeviceId: *deviceUuid,
	})

	currentUser.SetSession(w, loginInfo.UserId, loginInfo.RememberMe, realm.Name, sessionToken)

	http.Redirect(w, r, loginInfo.OriginalUrl, http.StatusFound)
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

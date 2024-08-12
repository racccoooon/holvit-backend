package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/services"
	"net/http"
)

type VerifyDeviceRequest struct {
	Token string `json:"token"`
	Code  string `json:"code"`
}

func VerifyDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var request VerifyDeviceRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		rcs.Error(err)
		return
	}

	tokenService := ioc.Get[services.TokenService](scope)
	loginInfo := tokenService.PeekLoginCode(ctx, request.Token).UnwrapErr(httpErrors.BadRequest().WithMessage("token not found"))

	currentUser := ioc.Get[services.CurrentSessionService](scope)
	deviceIdString, err := currentUser.DeviceIdString()
	if err != nil {
		rcs.Error(err)
		return
	}
	if deviceIdString != loginInfo.DeviceId {
		rcs.Error(httpErrors.Unauthorized().WithMessage("wrong device id"))
	}

	currentStep := loginInfo.NextStep
	if currentStep != constants.AuthenticateStepVerifyDevice {
		rcs.Error(httpErrors.Unauthorized().WithMessage(
			fmt.Sprintf("wrong login step '%s', expected '%s'", currentStep, constants.AuthenticateStepVerifyDevice)))
		return
	}

	deviceService := ioc.Get[services.DeviceService](scope)
	deviceService.AddKnownDevice(ctx, services.AddDeviceRequest{
		UserId:   loginInfo.UserId,
		DeviceId: deviceIdString,
	})

	nextStep, err := getNextStep(ctx, currentStep, &loginInfo)
	if err != nil {
		rcs.Error(err)
		return
	}
	err = nextStep.Prepare(ctx, &loginInfo)
	if err != nil {
		rcs.Error(err)
		return
	}

	loginInfo.NextStep = nextStep.Name()

	tokenService.OverwriteLoginCode(ctx, request.Token, loginInfo).SetErr(httpErrors.BadRequest().WithMessage("token not found")).Unwrap()

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err = encoder.Encode(VerifyLoginStepResponse{
		NextStep: loginInfo.NextStep,
	})
	if err != nil {
		rcs.Error(err)
		return
	}
}

type VerifyDeviceStep struct {
}

func (s *VerifyDeviceStep) Name() string {
	return constants.AuthenticateStepVerifyDevice
}

func (s *VerifyDeviceStep) NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error) {
	scope := middlewares.GetScope(ctx)

	deviceService := ioc.Get[services.DeviceService](scope)
	response := deviceService.IsKnownUserDevice(ctx, services.IsKnownDeviceRequest{
		UserId:   info.UserId,
		DeviceId: info.DeviceId,
	})

	return response.Id.IsNone() && response.RequiresVerification, nil
}

func (s *VerifyDeviceStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}

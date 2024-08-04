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

type VerifyTotpRequest struct {
	Token string `json:"token"`
	Code  string `json:"code"`
}

func VerifyTotp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var request VerifyTotpRequest
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
	if currentStep != constants.AuthenticateStepVerifyTotp {
		rcs.Error(httpErrors.Unauthorized().WithMessage(
			fmt.Sprintf("wrong login step '%s', expected '%s'", currentStep, constants.AuthenticateStepVerifyTotp)))
		return
	}

	userService := ioc.Get[services.UserService](scope)
	err = userService.VerifyTotp(ctx, services.VerifyTotpRequest{
		UserId: loginInfo.UserId,
		Code:   request.Code,
	})
	if err != nil {
		rcs.Error(err)
		return
	}

	nextStep, err := getNextStep(ctx, currentStep, loginInfo)
	if err != nil {
		rcs.Error(err)
		return
	}
	err = nextStep.Prepare(ctx, loginInfo)
	if err != nil {
		rcs.Error(err)
		return
	}

	err = tokenService.OverwriteLoginCode(ctx, request.Token, *loginInfo)
	if err != nil {
		rcs.Error(err)
		return
	}

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

type VerifyTotpStep struct {
}

func (s *VerifyTotpStep) Name() string {
	return constants.AuthenticateStepVerifyTotp
}

func (s *VerifyTotpStep) NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error) {
	scope := middlewares.GetScope(ctx)

	userService := ioc.Get[services.UserService](scope)
	return userService.RequiresTotp(ctx, info.UserId)
}

func (s *VerifyTotpStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}

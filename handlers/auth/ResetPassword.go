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

type ResetPasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
	Token       string `json:"token"`
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
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
	if currentStep != constants.AuthenticateStepResetPassword {
		rcs.Error(httpErrors.Unauthorized().WithMessage(
			fmt.Sprintf("wrong login step '%s', expected '%s'", currentStep, constants.AuthenticateStepVerifyPassword)))
		return
	}

	userService := ioc.Get[services.UserService](scope)
	err = userService.SetPassword(ctx, services.SetPasswordRequest{
		UserId:    loginInfo.UserId,
		Password:  request.NewPassword,
		Temporary: false,
	}, services.DangerousNoAuthStrategy{})
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

	loginInfo.NextStep = nextStep.Name()

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

type ResetPasswordStep struct{}

func (s *ResetPasswordStep) Name() string {
	return constants.AuthenticateStepResetPassword
}

func (s *ResetPasswordStep) NeedsToRun(ctx context.Context, loginInfo *services.LoginInfo) (bool, error) {
	scope := middlewares.GetScope(ctx)

	userService := ioc.Get[services.UserService](scope)
	isTemporary := userService.IsPasswordTemporary(ctx, loginInfo.UserId)

	return isTemporary, nil
}

func (s *ResetPasswordStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}

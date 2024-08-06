package auth

import (
	"encoding/json"
	"fmt"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
	"holvit/services"
	"net/http"
)

type VerifyPasswordRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Token      string `json:"token"`
	RememberMe bool   `json:"remember_me"`
}

func VerifyPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	currentUserService := ioc.Get[services.CurrentUserService](scope)
	deviceIdString, err := currentUserService.DeviceIdString()
	if err != nil {
		rcs.Error(err)
		return
	}

	var request VerifyPasswordRequest
	err = json.NewDecoder(r.Body).Decode(&request)
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

	currentStep := loginInfo.NextStep
	if currentStep != constants.AuthenticateStepVerifyPassword {
		rcs.Error(httpErrors.Unauthorized().WithMessage(
			fmt.Sprintf("wrong login step '%s', expected '%s'", currentStep, constants.AuthenticateStepVerifyPassword)))
		return
	}

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, loginInfo.RealmId).Unwrap()

	if request.RememberMe && !realm.EnableRememberMe {
		rcs.Error(httpErrors.BadRequest().WithMessage("realm does not allow remember me"))
		return
	}

	userService := ioc.Get[services.UserService](scope)
	loginResponse := userService.VerifyLogin(ctx, services.VerifyLoginRequest{
		Username: request.Username,
		Password: request.Password,
		RealmId:  loginInfo.RealmId,
	})

	loginInfo.UserId = loginResponse.UserId
	loginInfo.DeviceId = deviceIdString
	loginInfo.RememberMe = request.RememberMe

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

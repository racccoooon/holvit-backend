package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"holvit/config"
	"holvit/constants"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/services"
	"holvit/utils"
	"net/http"
)

type TotpOnboardingRequest struct {
	Token       string  `json:"token"`
	Code        string  `json:"code"`
	DisplayName *string `json:"display_name"`
}

func TotpOnboarding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var request TotpOnboardingRequest
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
	if currentStep != constants.AuthenticateStepTotpOnboarding {
		rcs.Error(httpErrors.Unauthorized().WithMessage(
			fmt.Sprintf("wrong login step '%s', expected '%s'", currentStep, constants.AuthenticateStepTotpOnboarding)))
		return
	}

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	isValid, err := totp.ValidateCustom(request.Code, loginInfo.EncryptedTotpOnboardingSecretBase64, now, totp.ValidateOpts{
		Period:    config.C.Totp.Period,
		Skew:      config.C.Totp.Skew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		rcs.Error(err)
		return
	}
	if !isValid {
		rcs.Error(httpErrors.Unauthorized().WithMessage("invalid totp code"))
		return
	}

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		rcs.Error(err)
		return
	}

	encryptedSecret, err := base64.StdEncoding.DecodeString(loginInfo.EncryptedTotpOnboardingSecretBase64)
	if err != nil {
		rcs.Error(err)
		return
	}

	totpSecret, err := utils.DecryptSymmetric(encryptedSecret, key)
	if err != nil {
		rcs.Error(err)
		return
	}

	userService := ioc.Get[services.UserService](scope)
	err = userService.AddTotp(ctx, services.AddTotpRequest{
		UserId:      loginInfo.UserId,
		DisplayName: h.FromPtr(request.DisplayName),
		Secret:      totpSecret,
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

type TotpOnboardingStep struct {
}

func (s *TotpOnboardingStep) Name() string {
	return constants.AuthenticateStepTotpOnboarding
}

func (s *TotpOnboardingStep) NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error) {
	scope := middlewares.GetScope(ctx)

	userService := ioc.Get[services.UserService](scope)
	requiresTotpOnboarding := userService.RequiresTotpOnboarding(ctx, info.UserId)
	if requiresTotpOnboarding.IsOk() {
		return requiresTotpOnboarding.Unwrap(), nil
	}
	return false, requiresTotpOnboarding.UnwrapErr()
}

func (s *TotpOnboardingStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	secret, err := utils.GenerateRandomBytes(constants.TotpSecretLength)
	if err != nil {
		return err
	}

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return err
	}

	encryptedSecret, err := utils.EncryptSymmetric(secret, key)
	if err != nil {
		return err
	}

	info.EncryptedTotpOnboardingSecretBase64 = base64.StdEncoding.EncodeToString(encryptedSecret)

	return nil
}

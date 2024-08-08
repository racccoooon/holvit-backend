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
	DisplayName *string `json:"displayName"`
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
	loginInfo := tokenService.PeekLoginCode(ctx, request.Token).UnwrapErr(httpErrors.Unauthorized().WithMessage("token not found"))

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

	key := config.C.GetSymmetricEncryptionKey()
	encryptedSecret, err := base64.StdEncoding.DecodeString(loginInfo.EncryptedTotpOnboardingSecretBase64)
	if err != nil {
		rcs.Error(err)
		return
	}
	totpSecret := utils.DecryptSymmetric(encryptedSecret, key)

	userService := ioc.Get[services.UserService](scope)
	userService.AddTotp(ctx, services.AddTotpRequest{
		UserId:      loginInfo.UserId,
		DisplayName: h.FromPtr(request.DisplayName),
		Secret:      totpSecret,
	}, services.DangerousNoAuthStrategy{})

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

	tokenService.OverwriteLoginCode(ctx, request.Token, loginInfo).SetErr(httpErrors.Unauthorized().WithMessage("token not found")).Unwrap()

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
	return requiresTotpOnboarding, nil
}

func (s *TotpOnboardingStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	secret, err := utils.GenerateRandomBytes(constants.TotpSecretLength)
	if err != nil {
		return err
	}

	key := config.C.GetSymmetricEncryptionKey()
	encryptedSecret := utils.EncryptSymmetric(secret, key)

	info.EncryptedTotpOnboardingSecretBase64 = base64.StdEncoding.EncodeToString(encryptedSecret)

	return nil
}

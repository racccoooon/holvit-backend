package auth

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"holvit/config"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/services"
	"holvit/utils"
	"net/http"
)

type OnboardingTotpRequest struct {
	Token string `json:"token"`
}

type OnboardingTotpResponse struct {
	SecretBase32 string `json:"secretBase32"`
}

func GetOnboardingTotp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var request OnboardingTotpRequest
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

	key := config.C.GetSymmetricEncryptionKey()
	encryptedSecret, err := base64.StdEncoding.DecodeString(loginInfo.EncryptedTotpOnboardingSecretBase64)
	if err != nil {
		rcs.Error(err)
		return
	}
	secret := utils.DecryptSymmetric(encryptedSecret, key)

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err = encoder.Encode(OnboardingTotpResponse{
		SecretBase32: base32.StdEncoding.EncodeToString(secret),
	})
	if err != nil {
		rcs.Error(err)
		return
	}
}

package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/requestContext"
	"holvit/services"
	"holvit/utils"
	"net/http"
	"strings"
)

func login(w http.ResponseWriter, r *http.Request, realmName string, request services.AuthorizationRequest) error {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	if err := r.ParseForm(); err != nil {
		return err
	}

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realms, _, err := realmRepository.FindRealms(ctx, repositories.RealmFilter{
		Name: &realmName,
	})
	if err != nil {
		return err
	}
	realm := realms[0]

	tokenService := ioc.Get[services.TokenService](scope)
	loginToken, err := tokenService.StoreLoginCode(ctx, services.LoginInfo{
		RealmId: realm.Id,
		Request: request,
	})
	if err != nil {
		return err
	}

	frontendData := utils.AuthFrontendData{
		Mode: constants.FrontendModeAuthenticate,
		Authenticate: &utils.AuthFrontendDataAuthenticate{
			Token:         loginToken,
			UseRememberMe: realm.EnableRememberMe,
		},
	}

	err = utils.ServeAuthFrontend(w, frontendData)
	if err != nil {
		return err
	}

	return nil
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	routeParams := mux.Vars(r)
	realmName := routeParams["realmName"]

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}
	request := services.AuthorizationRequest{
		ResponseTypes: strings.Split(r.Form.Get("response_type"), " "),
		RealmName:     realmName,
		ClientId:      r.Form.Get("client_id"),
		RedirectUri:   r.Form.Get("redirect_uri"),
		Scopes:        strings.Split(r.Form.Get("scope"), " "),
		State:         r.Form.Get("state"),
		ResponseMode:  r.Form.Get("response_mode"),
	}

	currentUserService := ioc.Get[services.CurrentUserService](scope)

	if currentUserService.GetCurrentUser() == nil {
		err := login(w, r, realmName, request)
		if err != nil {
			rcs.Error(err)
			return
		}
		return
	}

	oidcService := ioc.Get[services.OidcService](scope)
	response, err := oidcService.Authorize(ctx, request)
	if err != nil {
		rcs.Error(err)
		return
	}

	response.HandleHttp(w, r)
}

func AuthorizeGrant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	grants := r.Form["grant"]
	token := r.Form.Get("token")

	if len(grants) == 0 {
		rcs.Error(httpErrors.BadRequest().WithMessage("Missing grants"))
		return
	}
	if token == "" {
		rcs.Error(httpErrors.BadRequest().WithMessage("Missing token"))
		return
	}

	tokenService := ioc.Get[services.TokenService](scope)
	info, err := tokenService.RetrieveGrantInfo(ctx, token)
	if err != nil {
		rcs.Error(err)
		return
	}

	oidcService := ioc.Get[services.OidcService](scope)
	response, err := oidcService.Grant(ctx, services.GrantRequest{
		ClientId:             info.ClientId,
		RealmId:              info.RealmId,
		ScopeNames:           grants,
		AuthorizationRequest: info.AuthorizationRequest,
	})
	if err != nil {
		rcs.Error(err)
		return
	}

	response.HandleHttp(w, r)
}

func Token(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	grantType := r.Form.Get("grant_type")

	clientId, clientSecret, _ := r.BasicAuth()

	oidcService := ioc.Get[services.OidcService](scope)

	var response *services.TokenResponse
	var err error

	switch grantType {
	case constants.TokenGrantTypeAuthorizationCode:
		response, err = oidcService.HandleAuthorizationCode(ctx, services.AuthorizationCodeTokenRequest{
			RedirectUri:  r.Form.Get("redirect_uri"),
			Code:         r.Form.Get("code"),
			ClientId:     clientId,
			ClientSecret: clientSecret,
		})
		break
	case constants.TokenGrantTypeRefreshToken:
		response, err = oidcService.HandleRefreshToken(ctx, services.RefreshTokenRequest{
			RefreshToken: r.Form.Get("refresh_token"),
			ClientId:     clientId,
			ClientSecret: clientSecret,
			ScopeNames:   strings.Split(r.Form.Get("scope"), " "),
		})
		break
	default:
		rcs.Error(httpErrors.BadRequest().WithMessage(fmt.Sprintf("Unsupported grant_type '%s'", grantType)))
	}

	if err != nil {
		rcs.Error(err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		rcs.Error(err)
		return
	}

}

func UserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	if err := r.ParseForm(); err != nil {
		rcs.Error(err)
		return
	}

	bearer := r.Header.Get("Authorization")

	oidcService := ioc.Get[services.OidcService](scope)
	response := oidcService.UserInfo(bearer)

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		return
	}
}

type VerifyTotpRequest struct {
	Token    string `json:"token"`
	Code     string `json:"code"`
	DeviceId string `json:"device_id"`
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

	if !loginInfo.PasswordVerified {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	if !loginInfo.TotpVerified {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	if loginInfo.DeviceVerified {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	if loginInfo.DeviceVerificationCode != request.Code {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	if loginInfo.DeviceId != request.DeviceId {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	deviceService := ioc.Get[services.DeviceService](scope)
	err = deviceService.AddKnownDevice(ctx, services.AddDeviceRequest{
		UserId:    loginInfo.UserId,
		DeviceId:  loginInfo.DeviceId,
		UserAgent: r.UserAgent(),
		Ip:        utils.GetRequestIp(r),
	})
	if err != nil {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err = encoder.Encode(VerifyLoginStepResponse{
		Success:     true,
		RequireTotp: false,
		NewDevice:   false,
	})
	if err != nil {
		rcs.Error(err)
		return
	}
}

type VerifyDeviceRequest struct {
	Token    string `json:"token"`
	DeviceId string `json:"device_id"`
	Code     string `json:"code"`
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
	loginInfo, err := tokenService.PeekLoginCode(ctx, request.Token)
	if err != nil {
		rcs.Error(err)
		return
	}

	if !loginInfo.PasswordVerified {
		rcs.Error(httpErrors.Unauthorized())
		return
	}

	if loginInfo.TotpVerified {
		rcs.Error(httpErrors.Unauthorized())
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

	loginInfo.TotpVerified = true

	deviceVerificationRequired, err := doesDeviceRequireVerification(r, loginInfo.UserId, request.DeviceId, loginInfo)
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
		Success:     true,
		RequireTotp: false,
		NewDevice:   deviceVerificationRequired,
	})
	if err != nil {
		rcs.Error(err)
		return
	}
}

type VerifyPasswordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
	DeviceId string `json:"device_id"`
}

type VerifyLoginStepResponse struct {
	Success     bool `json:"success"`
	RequireTotp bool `json:"require_totp"`
	NewDevice   bool `json:"new_device"`
}

func VerifyPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var request VerifyPasswordRequest
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

	userService := ioc.Get[services.UserService](scope)
	loginResponse, err := userService.VerifyLogin(ctx, services.VerifyLoginRequest{
		UsernameOrEmail: request.Username,
		Password:        request.Password,
		RealmId:         loginInfo.RealmId,
	})
	if err != nil {
		rcs.Error(err)
		return
	}

	loginInfo.UserId = loginResponse.UserId
	loginInfo.PasswordVerified = true
	loginInfo.TotpVerified = !loginResponse.RequireTotp

	deviceVerificationRequired, err := doesDeviceRequireVerification(r, loginResponse.UserId, request.DeviceId, loginInfo)
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
		Success:     true,
		RequireTotp: loginResponse.RequireTotp,
		NewDevice:   deviceVerificationRequired,
	})
	if err != nil {
		rcs.Error(err)
		return
	}
}

func doesDeviceRequireVerification(r *http.Request, userId uuid.UUID, deviceId string, loginInfo *services.LoginInfo) (bool, error) {
	//TODO: validate request parameters (device id must be a uuid)

	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	deviceService := ioc.Get[services.DeviceService](scope)
	isKnownUserDevice, err := deviceService.IsKnownUserDevice(ctx, services.IsKnownDeviceRequest{
		UserId:   userId,
		DeviceId: deviceId,
	})
	if err != nil {
		return false, err
	}

	loginInfo.DeviceVerified = isKnownUserDevice.IsKnown || !isKnownUserDevice.RequiresVerification

	if !loginInfo.DeviceVerified {
		deviceCode, err := deviceService.SendVerificationEmail(ctx, services.SendVerificationRequest{
			UserId:    userId,
			DeviceId:  deviceId,
			UserAgent: r.UserAgent(),
		})
		if err != nil {
			return false, err
		}

		loginInfo.DeviceId = deviceId
		loginInfo.DeviceVerificationCode = deviceCode.Code
	}

	return !loginInfo.DeviceVerified, nil
}

func Jwks(w http.ResponseWriter, r *http.Request) {
}

func EndSession(w http.ResponseWriter, r *http.Request) {
}

func Discovery(w http.ResponseWriter, r *http.Request) {
}

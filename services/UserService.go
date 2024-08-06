package services

import (
	"context"
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"holvit/config"
	"holvit/constants"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/utils"
)

type AuthStrategy interface {
	Authorize(ctx context.Context, userId uuid.UUID) // TODO: maybe return an error?
}

type DangerousNoAuthStrategy struct{}

func (DangerousNoAuthStrategy) Authorize(ctx context.Context, userId uuid.UUID) {}

type TotpAuthStrategy struct {
	Code string
}

func (s TotpAuthStrategy) Authorize(ctx context.Context, userId uuid.UUID) {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, userId).Unwrap()

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credentials := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		UserId: h.Some(user.Id),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Unwrap()
	if !credentials.Any() {
		panic(httpErrors.Unauthorized().WithMessage("no totp configured for user"))
	}

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		panic(err)
	}

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	for _, credential := range credentials.Values() {
		details := credential.Details.(repos.CredentialTotpDetails)

		encryptedSecret, err := base64.StdEncoding.DecodeString(details.EncryptedSecretBase64)
		if err != nil {
			panic(err)
		}

		secret, err := utils.DecryptSymmetric(encryptedSecret, key)
		if err != nil {
			panic(err)
		}

		isValid, err := totp.ValidateCustom(s.Code, string(secret), now, totp.ValidateOpts{
			Period:    config.C.Totp.Period,
			Skew:      config.C.Totp.Skew,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil {
			panic(err)
		}

		if isValid {
			// we found a matching totp for the user
			return
		}
	}
}

type PasswordAuthStrategy struct {
	Password string
}

func (s PasswordAuthStrategy) Authorize(ctx context.Context, userId uuid.UUID) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(userId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).Unwrap().FirstOrNone().UnwrapErr(httpErrors.Unauthorized().WithMessage("no password for user"))

	details := credential.Details.(repos.CredentialPasswordDetails)

	err := utils.CompareHash(s.Password, details.HashedPassword)
	if err != nil {
		panic(httpErrors.Unauthorized().WithMessage("failed to verify password"))
	}
}

type CreateUserRequest struct {
	RealmId uuid.UUID

	Username string
	Email    h.Optional[string]
}

type SetPasswordRequest struct {
	UserId    uuid.UUID
	Password  string
	Temporary bool
}

type VerifyLoginRequest struct {
	Username string
	Password string
	RealmId  uuid.UUID
}

type VerifyLoginResponse struct {
	UserId uuid.UUID
}

type VerifyTotpRequest struct {
	UserId uuid.UUID
	Code   string
}

type AddTotpRequest struct {
	UserId      uuid.UUID
	Secret      []byte
	DisplayName h.Optional[string]
}

type UserService interface {
	CreateUser(ctx context.Context, request CreateUserRequest) h.Result[uuid.UUID]

	SetPassword(ctx context.Context, request SetPasswordRequest, strategy AuthStrategy) error
	IsPasswordTemporary(ctx context.Context, userId uuid.UUID) bool

	AddTotp(ctx context.Context, request AddTotpRequest, strategy AuthStrategy) error
	RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) h.Result[bool]
	HasTotpConfigured(ctx context.Context, userId uuid.UUID) bool

	VerifyLogin(ctx context.Context, request VerifyLoginRequest) VerifyLoginResponse
	VerifyTotp(ctx context.Context, request VerifyTotpRequest)
}

type UserServiceImpl struct {
}

func NewUserService() UserService {
	return &UserServiceImpl{}
}

func (u *UserServiceImpl) CreateUser(ctx context.Context, request CreateUserRequest) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)

	return userRepository.CreateUser(ctx, &repos.User{
		RealmId:  request.RealmId,
		Username: request.Username,
		Email:    request.Email,
	})
}

func (u *UserServiceImpl) SetPassword(ctx context.Context, request SetPasswordRequest, strategy AuthStrategy) error {
	strategy.Authorize(ctx, request.UserId)

	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(request.UserId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).Unwrap().SingleOrNone()

	if existingCredential, ok := credential.Get(); ok {
		err := credentialRepository.DeleteCredential(ctx, existingCredential.Id)
		if err != nil {
			return err
		}
	}

	hashAlgorithm := config.C.GetHashAlgorithm()
	hashed, err := hashAlgorithm.Hash(request.Password)
	if err != nil {
		return err
	}

	err = credentialRepository.CreateCredential(ctx, &repos.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypePassword,
		Details: repos.CredentialPasswordDetails{
			HashedPassword: hashed,
			Temporary:      request.Temporary,
		},
	}).UnwrapErr()
	if err != nil {
		return err
	}

	return nil
}

func (u *UserServiceImpl) IsPasswordTemporary(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(userId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).Unwrap().First()

	return credential.Details.(repos.CredentialPasswordDetails).Temporary
}

func (u *UserServiceImpl) AddTotp(ctx context.Context, request AddTotpRequest, strategy AuthStrategy) error {
	strategy.Authorize(ctx, request.UserId)

	scope := middlewares.GetScope(ctx)

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return err
	}

	encryptedSecret, err := utils.EncryptSymmetric(request.Secret, key)
	if err != nil {
		return err
	}

	encryptedSecretBase64 := base64.StdEncoding.EncodeToString(encryptedSecret)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	err = credentialRepository.CreateCredential(ctx, &repos.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypeTotp,
		Details: repos.CredentialTotpDetails{
			DisplayName:           request.DisplayName.OrDefault("New Totp"),
			EncryptedSecretBase64: encryptedSecretBase64,
		},
	}).UnwrapErr()
	if err != nil {
		return err
	}

	return nil
}

func (u *UserServiceImpl) RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) h.Result[bool] {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	anyTotp := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: h.Some(repos.NewPagingInfo(1, 0)),
		},
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Unwrap().Any()

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, userId).Unwrap()

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, user.RealmId).Unwrap()

	return h.Ok(!anyTotp && realm.RequireTotp)
}

func (u *UserServiceImpl) HasTotpConfigured(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	return credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: h.Some(repos.NewPagingInfo(1, 0)),
		},
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Unwrap().Any()
}

func (u *UserServiceImpl) VerifyLogin(ctx context.Context, request VerifyLoginRequest) VerifyLoginResponse {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUsers(ctx, repos.UserFilter{
		RealmId:  h.Some(request.RealmId),
		Username: h.Some(request.Username),
	}).Unwrap().First()

	PasswordAuthStrategy{
		Password: request.Password,
	}.Authorize(ctx, user.Id)

	return VerifyLoginResponse{
		UserId: user.Id,
	}
}

func (u *UserServiceImpl) VerifyTotp(ctx context.Context, request VerifyTotpRequest) {
	TotpAuthStrategy{
		Code: request.Code,
	}.Authorize(ctx, request.UserId)
}

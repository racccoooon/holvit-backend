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
	Authorize(ctx context.Context, userId uuid.UUID) bool // TODO: maybe return an error?
}

type DangerousNoAuthStrategy struct{}

func (DangerousNoAuthStrategy) Authorize(ctx context.Context, userId uuid.UUID) bool {
	return true
}

type TotpAuthStrategy struct {
	Code string
}

func (s TotpAuthStrategy) Authorize(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credentials := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	})
	if !credentials.Any() {
		panic(httpErrors.Unauthorized().WithMessage("no totp configured for user"))
	}

	key := config.C.GetSymmetricEncryptionKey()

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	for _, credential := range credentials.Values() {
		details := credential.Details.(repos.CredentialTotpDetails)

		encryptedSecret, err := base64.StdEncoding.DecodeString(details.EncryptedSecretBase64)
		if err != nil {
			panic(err)
		}

		secret := utils.DecryptSymmetric(encryptedSecret, key)

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
			return true
		}
	}
	return false
}

type PasswordAuthStrategy struct {
	Password string
}

func (s PasswordAuthStrategy) Authorize(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(userId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).SingleOrNone()

	if credential, ok := credential.Get(); ok {
		details := credential.Details.(repos.CredentialPasswordDetails)

		result := utils.ValidateHash(s.Password, details.HashedPassword, config.C.GetHasher())
		if result.IsValid && result.NeedsRehash {
			// TODO: rehash password!
		}
		return result.IsValid
	}
	return false
}

type CreateUserRequest struct {
	RealmId uuid.UUID

	Username string
	Email    h.Opt[string]
}

type SetPasswordRequest struct {
	UserId    uuid.UUID
	Password  string
	Temporary bool
}

type VerifyLoginRequest struct {
	Username string
	UserId   uuid.UUID
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
	DisplayName h.Opt[string]
}

type UserService interface {
	CreateUser(ctx context.Context, request CreateUserRequest) h.Result[uuid.UUID]

	SetPassword(ctx context.Context, request SetPasswordRequest, strategy AuthStrategy)
	IsPasswordTemporary(ctx context.Context, userId uuid.UUID) bool

	AddTotp(ctx context.Context, request AddTotpRequest, strategy AuthStrategy)
	RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) bool
	HasTotpConfigured(ctx context.Context, userId uuid.UUID) bool

	VerifyLogin(ctx context.Context, request VerifyLoginRequest) VerifyLoginResponse
	VerifyTotp(ctx context.Context, request VerifyTotpRequest)
}

type userServiceImpl struct {
}

func NewUserService() UserService {
	return &userServiceImpl{}
}

func (u *userServiceImpl) CreateUser(ctx context.Context, request CreateUserRequest) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)

	return userRepository.CreateUser(ctx, repos.User{
		RealmId:  request.RealmId,
		Username: request.Username,
		Email:    request.Email,
	})
}

func (u *userServiceImpl) SetPassword(ctx context.Context, request SetPasswordRequest, strategy AuthStrategy) {
	if !strategy.Authorize(ctx, request.UserId) {
		//TODO: panic or return error
	}

	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(request.UserId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).SingleOrNone()

	if existingCredential, ok := credential.Get(); ok {
		credentialRepository.DeleteCredential(ctx, existingCredential.Id)
	}

	hashAlgorithm := config.C.GetHasher()
	hashed := hashAlgorithm.Hash(request.Password)

	credentialRepository.CreateCredential(ctx, repos.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypePassword,
		Details: repos.CredentialPasswordDetails{
			HashedPassword: hashed,
			Temporary:      request.Temporary,
		},
	}).Unwrap()
}

func (u *userServiceImpl) IsPasswordTemporary(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(userId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).Single()

	return credential.Details.(repos.CredentialPasswordDetails).Temporary
}

func (u *userServiceImpl) AddTotp(ctx context.Context, request AddTotpRequest, strategy AuthStrategy) {
	if !strategy.Authorize(ctx, request.UserId) {
		//TODO:
	}

	scope := middlewares.GetScope(ctx)

	key := config.C.GetSymmetricEncryptionKey()
	encryptedSecret := utils.EncryptSymmetric(request.Secret, key)
	encryptedSecretBase64 := base64.StdEncoding.EncodeToString(encryptedSecret)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	_ = credentialRepository.CreateCredential(ctx, repos.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypeTotp,
		Details: repos.CredentialTotpDetails{
			DisplayName:           request.DisplayName.OrDefault("New Totp"),
			EncryptedSecretBase64: encryptedSecretBase64,
		},
	}).Unwrap()
}

func (u *userServiceImpl) RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	anyTotp := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: h.Some(repos.NewPagingInfo(1, 1)),
		},
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Any()

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, userId).Unwrap()

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, user.RealmId).Unwrap()

	return !anyTotp && realm.RequireTotp
}

func (u *userServiceImpl) HasTotpConfigured(ctx context.Context, userId uuid.UUID) bool {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	return credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: h.Some(repos.NewPagingInfo(1, 1)),
		},
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Any()
}

// TODO: refactor looking up user by name and verifying password
func (u *userServiceImpl) VerifyLogin(ctx context.Context, request VerifyLoginRequest) VerifyLoginResponse {
	isValid := PasswordAuthStrategy{
		Password: request.Password,
	}.Authorize(ctx, request.UserId)

	if !isValid {
		// TODO: also do this for all other authroize things
		panic(httpErrors.Unauthorized().WithMessage("invalid username or password"))
	}

	return VerifyLoginResponse{
		UserId: request.UserId,
	}
}

// TODO: refactor this away(just use the totp auth strategy where needed)
func (u *userServiceImpl) VerifyTotp(ctx context.Context, request VerifyTotpRequest) {
	isValid := TotpAuthStrategy{
		Code: request.Code,
	}.Authorize(ctx, request.UserId)
	if !isValid {
		panic(httpErrors.Unauthorized().WithMessage("invalid totp code"))
	}
}

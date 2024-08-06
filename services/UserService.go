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

type CreateUserRequest struct {
	RealmId uuid.UUID

	Username *string
	Email    *string
}

type CreateUserResponse struct {
	Id uuid.UUID
}

type SetPasswordRequest struct {
	UserId      uuid.UUID
	Password    string
	OldPassword *string
	Temporary   bool
}

type SetPasswordInitialRequest struct {
	UserId    uuid.UUID
	Password  string
	Temporary bool
}

type VerifyLoginRequest struct {
	UsernameOrEmail string
	Password        string
	RealmId         uuid.UUID
}

type VerifyLoginResponse struct {
	UserId uuid.UUID
}

type VerifyTotpRequest struct {
	UserId uuid.UUID
	Code   string
}

type AddTotpRequest struct {
	UserId                uuid.UUID
	EncryptedSecretBase64 *string
	DisplayName           *string
}

type UserService interface {
	CreateUser(ctx context.Context, request CreateUserRequest) (*CreateUserResponse, error)
	SetPassword(ctx context.Context, request SetPasswordRequest) error
	VerifyLogin(ctx context.Context, request VerifyLoginRequest) (*VerifyLoginResponse, error)
	IsPasswordTemporary(ctx context.Context, userId uuid.UUID) (bool, error)
	RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) (bool, error)
	RequiresTotp(ctx context.Context, userId uuid.UUID) (bool, error)
	VerifyTotp(ctx context.Context, request VerifyTotpRequest) error
	AddTotp(ctx context.Context, request AddTotpRequest) error
}

type UserServiceImpl struct {
}

func NewUserService() UserService {
	return &UserServiceImpl{}
}

func (u *UserServiceImpl) CreateUser(ctx context.Context, request CreateUserRequest) (*CreateUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)

	userId := userRepository.CreateUser(ctx, &repos.User{
		RealmId:  request.RealmId,
		Username: h.FromPtr(request.Username),
		Email:    h.FromPtr(request.Email),
	}).Unwrap()

	return &CreateUserResponse{
		Id: userId,
	}, nil
}

func (u *UserServiceImpl) IsPasswordTemporary(ctx context.Context, userId uuid.UUID) (bool, error) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(userId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).Unwrap().FirstOrNone()
	if credential.IsNone() {
		return false, nil
	}

	return credential.Unwrap().Details.(repos.CredentialPasswordDetails).Temporary, nil
}

func setPassword(ctx context.Context, request SetPasswordRequest, verifyOld bool) error {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(request.UserId),
		Type:       h.Some(constants.CredentialTypePassword),
	}).Unwrap().SingleOrNone()

	if existingCredential, ok := credential.Get(); ok {

		if verifyOld {
			if request.OldPassword == nil {
				return httpErrors.Unauthorized().WithMessage("missing old password")
			}

			err := verifyPassword(utils.Ptr(existingCredential), *request.OldPassword)
			if err != nil {
				return err
			}
		}

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

	_ = credentialRepository.CreateCredential(ctx, &repos.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypePassword,
		Details: repos.CredentialPasswordDetails{
			HashedPassword: hashed,
			Temporary:      request.Temporary,
		},
	}).Unwrap()

	return nil
}

func (u *UserServiceImpl) SetPasswordDangerouslyWithoutVerifyingOldPassword(ctx context.Context, request SetPasswordRequest) error {
	return setPassword(ctx, request, false)
}

func (u *UserServiceImpl) SetPassword(ctx context.Context, request SetPasswordRequest) error {
	return setPassword(ctx, request, true)
}

func verifyPassword(credential *repos.Credential, password string) error {
	details := credential.Details.(repos.CredentialPasswordDetails)

	err := utils.CompareHash(password, details.HashedPassword)
	if err != nil {
		return httpErrors.Unauthorized().WithMessage("failed to verify password")
	}

	return nil
}

func (u *UserServiceImpl) VerifyLogin(ctx context.Context, request VerifyLoginRequest) (*VerifyLoginResponse, error) {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUsers(ctx, repos.UserFilter{
		RealmId:         h.Some(request.RealmId),
		UsernameOrEmail: h.Some(request.UsernameOrEmail),
	}).Unwrap().First()

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credential := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		UserId: h.Some(user.Id),
		Type:   h.Some(constants.CredentialTypePassword),
	}).Unwrap().First()

	err := verifyPassword(&credential, request.Password)
	if err != nil {
		return nil, err
	}

	return &VerifyLoginResponse{
		UserId: user.Id,
	}, nil
}

func (u *UserServiceImpl) VerifyTotp(ctx context.Context, request VerifyTotpRequest) error {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, request.UserId).Unwrap()

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credentials := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{},
		UserId:     h.Some(user.Id),
		Type:       h.Some(constants.CredentialTypeTotp),
	}).Unwrap()
	if credentials.Count() == 0 {
		return httpErrors.Unauthorized().WithMessage("no totp configured")
	}

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return err
	}

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	for _, credential := range credentials.Values() {
		details := credential.Details.(repos.CredentialTotpDetails)

		encryptedSecret, err := base64.StdEncoding.DecodeString(details.EncryptedSecretBase64)
		if err != nil {
			return err
		}

		secret, err := utils.DecryptSymmetric(encryptedSecret, key)
		if err != nil {
			return err
		}

		isValid, err := totp.ValidateCustom(request.Code, string(secret), now, totp.ValidateOpts{
			Period:    config.C.Totp.Period,
			Skew:      config.C.Totp.Skew,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil {
			return err
		}

		if isValid {
			// we found a matching totp for the user
			return nil
		}
	}

	return httpErrors.Unauthorized().WithMessage("no matching totp found")
}

func (u *UserServiceImpl) AddTotp(ctx context.Context, request AddTotpRequest) error {
	scope := middlewares.GetScope(ctx)

	encryptedSecretBase64 := request.EncryptedSecretBase64

	if encryptedSecretBase64 == nil {
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

		encryptedSecretBase64 = utils.Ptr(base64.StdEncoding.EncodeToString(encryptedSecret))
	}

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	_ = credentialRepository.CreateCredential(ctx, &repos.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypeTotp,
		Details: repos.CredentialTotpDetails{
			DisplayName:           utils.GetOrDefault(request.DisplayName, "New Totp"),
			EncryptedSecretBase64: *encryptedSecretBase64,
		},
	}).Unwrap()

	return nil
}

func (u *UserServiceImpl) RequiresTotp(ctx context.Context, userId uuid.UUID) (bool, error) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credentials := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: h.Some(repos.NewPagingInfo(1, 0)),
		},
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Unwrap()

	return credentials.Count() > 0, nil
}

func (u *UserServiceImpl) RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) (bool, error) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repos.CredentialRepository](scope)

	credentials := credentialRepository.FindCredentials(ctx, repos.CredentialFilter{
		BaseFilter: repos.BaseFilter{
			PagingInfo: h.Some(repos.PagingInfo{
				PageSize:   1,
				PageNumber: 0,
			}),
		},
		UserId: h.Some(userId),
		Type:   h.Some(constants.CredentialTypeTotp),
	}).Unwrap()

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, userId).Unwrap()

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, user.RealmId).Unwrap()

	return credentials.Count() == 0 && realm.RequireTotp, nil
}

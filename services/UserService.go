package services

import (
	"context"
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"holvit/config"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
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

	userRepository := ioc.Get[repositories.UserRepository](scope)

	userId, err := userRepository.CreateUser(ctx, &repositories.User{
		RealmId:  request.RealmId,
		Username: request.Username,
		Email:    request.Email,
	})
	if err != nil {
		return nil, err
	}

	return &CreateUserResponse{
		Id: userId,
	}, nil
}

func (u *UserServiceImpl) IsPasswordTemporary(ctx context.Context, userId uuid.UUID) (bool, error) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	credentials, count, err := credentialRepository.FindCredentials(ctx, repositories.CredentialFilter{
		BaseFilter: repositories.BaseFilter{},
		UserId:     &userId,
		Type:       utils.Ptr(constants.CredentialTypePassword),
	})
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	credential := credentials[0]

	return credential.Details.(repositories.CredentialPasswordDetails).Temporary, nil
}

func (u *UserServiceImpl) SetPassword(ctx context.Context, request SetPasswordRequest) error {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	credentials, _, err := credentialRepository.FindCredentials(ctx, repositories.CredentialFilter{
		BaseFilter: repositories.BaseFilter{},
		UserId:     &request.UserId,
		Type:       utils.Ptr(constants.CredentialTypePassword),
	})
	if err != nil {
		return err
	}

	if credentials != nil {
		credential := credentials[0]

		err = verifyPassword(credential, request.Password)
		if err != nil {
			return err
		}

		err = credentialRepository.DeleteCredential(ctx, credential.Id)
		if err != nil {
			return err
		}
	}

	hashAlgorithm := config.C.GetHashAlgorithm()
	hashed, err := hashAlgorithm.Hash(request.Password)
	if err != nil {
		return err
	}

	_, err = credentialRepository.CreateCredential(ctx, &repositories.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypePassword,
		Details: repositories.CredentialPasswordDetails{
			HashedPassword: hashed,
			Temporary:      request.Temporary,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func verifyPassword(credential *repositories.Credential, password string) error {
	details := credential.Details.(repositories.CredentialPasswordDetails)

	err := utils.CompareHash(password, details.HashedPassword)
	if err != nil {
		return httpErrors.Unauthorized()
	}

	return nil
}

func (u *UserServiceImpl) VerifyLogin(ctx context.Context, request VerifyLoginRequest) (*VerifyLoginResponse, error) {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repositories.UserRepository](scope)
	users, count, err := userRepository.FindUsers(ctx, repositories.UserFilter{
		RealmId:            request.RealmId,
		UsernameOrPassword: &request.UsernameOrEmail,
	})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, httpErrors.Unauthorized()
	}
	user := users[0]

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	credentials, count, err := credentialRepository.FindCredentials(ctx, repositories.CredentialFilter{
		BaseFilter: repositories.BaseFilter{},
		UserId:     &user.Id,
		Type:       utils.Ptr(constants.CredentialTypePassword),
	})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, httpErrors.Unauthorized()
	}
	credential := credentials[0]

	err = verifyPassword(credential, request.Password)
	if err != nil {
		return nil, err
	}

	return &VerifyLoginResponse{
		UserId: user.Id,
	}, nil
}

func (u *UserServiceImpl) VerifyTotp(ctx context.Context, request VerifyTotpRequest) error {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repositories.UserRepository](scope)
	user, err := userRepository.FindUserById(ctx, request.UserId)
	if err != nil {
		return err
	}

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	credentials, count, err := credentialRepository.FindCredentials(ctx, repositories.CredentialFilter{
		BaseFilter: repositories.BaseFilter{},
		UserId:     &user.Id,
		Type:       utils.Ptr(constants.CredentialTypeTotp),
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return httpErrors.Unauthorized()
	}

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return err
	}

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	for _, credential := range credentials {
		details := credential.Details.(repositories.CredentialTotpDetails)

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

	return httpErrors.Unauthorized()
}

func (s *UserServiceImpl) AddTotp(ctx context.Context, request AddTotpRequest) error {
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

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	_, err := credentialRepository.CreateCredential(ctx, &repositories.Credential{
		UserId: request.UserId,
		Type:   constants.CredentialTypeTotp,
		Details: repositories.CredentialTotpDetails{
			DisplayName:           utils.GetOrDefault(request.DisplayName, "New Totp"),
			EncryptedSecretBase64: *encryptedSecretBase64,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *UserServiceImpl) RequiresTotp(ctx context.Context, userId uuid.UUID) (bool, error) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	_, totpCount, err := credentialRepository.FindCredentials(ctx, repositories.CredentialFilter{
		BaseFilter: repositories.BaseFilter{
			PagingInfo: repositories.PagingInfo{
				PageSize:   1,
				PageNumber: 0,
			},
		},
		UserId: &userId,
		Type:   utils.Ptr(constants.CredentialTypeTotp),
	})
	if err != nil {
		return false, err
	}

	return totpCount > 0, nil
}

func (s *UserServiceImpl) RequiresTotpOnboarding(ctx context.Context, userId uuid.UUID) (bool, error) {
	scope := middlewares.GetScope(ctx)

	credentialRepository := ioc.Get[repositories.CredentialRepository](scope)

	_, totpCount, err := credentialRepository.FindCredentials(ctx, repositories.CredentialFilter{
		BaseFilter: repositories.BaseFilter{
			PagingInfo: repositories.PagingInfo{
				PageSize:   1,
				PageNumber: 0,
			},
		},
		UserId: &userId,
		Type:   utils.Ptr(constants.CredentialTypeTotp),
	})
	if err != nil {
		return false, err
	}

	userRpository := ioc.Get[repositories.UserRepository](scope)
	user, err := userRpository.FindUserById(ctx, userId)
	if err != nil {
		return false, err
	}

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realm, err := realmRepository.FindRealmById(ctx, user.RealmId)
	if err != nil {
		return false, err
	}

	return totpCount == 0 && realm.RequireTotp, nil
}

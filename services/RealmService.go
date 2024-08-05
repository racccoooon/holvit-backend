package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/cache"
	"holvit/config"
	"holvit/constants"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/utils"
)

type CreateRealmRequest struct {
	Name        string
	DisplayName string

	RequireUsername           *bool
	RequireEmail              *bool
	RequireDeviceVerification *bool
	RequireTotp               *bool
	EnableRememberMe          *bool
}

type CreateRealmResponse struct {
	Id uuid.UUID
}

type RealmService interface {
	CreateRealm(ctx context.Context, request CreateRealmRequest) (*CreateRealmResponse, error)
	InitializeRealmKeys(ctx context.Context) error
}

type RealmServiceImpl struct{}

func NewRealmService() RealmService {
	return &RealmServiceImpl{}
}

func (s *RealmServiceImpl) CreateRealm(ctx context.Context, request CreateRealmRequest) (*CreateRealmResponse, error) {
	scope := middlewares.GetScope(ctx)

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return nil, err
	}

	privateKey, _, err := utils.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	privateKeyBytes := utils.ExportPrivateKey(privateKey)
	encryptedPrivateKeyBytes, err := utils.EncryptSymmetric(privateKeyBytes, key)
	if err != nil {
		return nil, err
	}

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realmId := realmRepository.CreateRealm(ctx, &repos.Realm{
		Name:                      request.Name,
		DisplayName:               request.DisplayName,
		EncryptedPrivateKey:       encryptedPrivateKeyBytes,
		RequireUsername:           utils.GetOrDefault(request.RequireUsername, true),
		RequireEmail:              utils.GetOrDefault(request.RequireUsername, false),
		RequireDeviceVerification: utils.GetOrDefault(request.RequireDeviceVerification, false),
		RequireTotp:               utils.GetOrDefault(request.RequireTotp, false),
		EnableRememberMe:          utils.GetOrDefault(request.EnableRememberMe, false),
	}).Unwrap()

	s.createOpenIdScope(ctx, realmId)
	s.createEmailScope(ctx, realmId)
	s.createProfileScope(ctx, realmId)

	keyCache := ioc.Get[cache.KeyCache](scope)
	keyCache.Set(realmId, privateKeyBytes)

	return &CreateRealmResponse{
		Id: realmId,
	}, nil
}

func (s *RealmServiceImpl) createProfileScope(ctx context.Context, realmId uuid.UUID) {
	scope := middlewares.GetScope(ctx)

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	claimMapperRepository := ioc.Get[repos.ClaimMapperRepository](scope)

	scopeId := scopeRepository.CreateScope(ctx, repos.Scope{
		RealmId:     realmId,
		Name:        "profile",
		DisplayName: "Profile",
		Description: "Access your name",
		SortIndex:   3,
	})

	mapperId := claimMapperRepository.CreateClaimMapper(ctx, &repos.ClaimMapper{
		BaseModel:   repos.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Username",
		Description: "The username of the user",
		Type:        constants.ClaimMapperUserInfo,
		Details: repos.UserInfoClaimMapperDetails{
			ClaimName: "preferred_username",
			Property:  constants.UserInfoPropertyUsername,
		},
	}).Unwrap()

	_ = claimMapperRepository.AssociateClaimMapper(ctx, repos.AssociateScopeClaimRequest{
		ClaimMapperId: mapperId,
		ScopeId:       scopeId.Unwrap(),
	}).Unwrap()
}

func (s *RealmServiceImpl) createEmailScope(ctx context.Context, realmId uuid.UUID) {
	scope := middlewares.GetScope(ctx)

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	claimMapperRepository := ioc.Get[repos.ClaimMapperRepository](scope)

	scopeId := scopeRepository.CreateScope(ctx, repos.Scope{
		RealmId:     realmId,
		Name:        "email",
		DisplayName: "Email",
		Description: "Access your email address",
		SortIndex:   2,
	}).Unwrap()

	mapperId := claimMapperRepository.CreateClaimMapper(ctx, &repos.ClaimMapper{
		BaseModel:   repos.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Email address",
		Description: "The primary email address of the user",
		Type:        constants.ClaimMapperUserInfo,
		Details: repos.UserInfoClaimMapperDetails{
			ClaimName: "email",
			Property:  constants.UserInfoPropertyEmail,
		},
	}).Unwrap()

	_ = claimMapperRepository.AssociateClaimMapper(ctx, repos.AssociateScopeClaimRequest{
		ClaimMapperId: mapperId,
		ScopeId:       scopeId,
	}).Unwrap()

	mapperId = claimMapperRepository.CreateClaimMapper(ctx, &repos.ClaimMapper{
		BaseModel:   repos.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Email address verified",
		Description: "Whether the email address was verified or not",
		Type:        constants.ClaimMapperUserInfo,
		Details: repos.UserInfoClaimMapperDetails{
			ClaimName: "email_verified",
			Property:  constants.UserInfoPropertyEmailVerified,
		},
	}).Unwrap()

	_ = claimMapperRepository.AssociateClaimMapper(ctx, repos.AssociateScopeClaimRequest{
		ClaimMapperId: mapperId,
		ScopeId:       scopeId,
	}).Unwrap()
}

func (s *RealmServiceImpl) createOpenIdScope(ctx context.Context, realmId uuid.UUID) {
	scope := middlewares.GetScope(ctx)

	scopeRepository := ioc.Get[repos.ScopeRepository](scope)
	claimMapperRepository := ioc.Get[repos.ClaimMapperRepository](scope)

	scopeId := scopeRepository.CreateScope(ctx, repos.Scope{
		RealmId:     realmId,
		Name:        "openid",
		DisplayName: "OpenId Connect",
		Description: "Sign you in",
		SortIndex:   1,
	}).Unwrap()

	mapperId := claimMapperRepository.CreateClaimMapper(ctx, &repos.ClaimMapper{
		BaseModel:   repos.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Subject Identifier",
		Description: "The id of the user",
		Type:        constants.ClaimMapperUserInfo,
		Details: repos.UserInfoClaimMapperDetails{
			ClaimName: "sub",
			Property:  constants.UserInfoPropertyId,
		},
	}).Unwrap()

	_ = claimMapperRepository.AssociateClaimMapper(ctx, repos.AssociateScopeClaimRequest{
		ClaimMapperId: mapperId,
		ScopeId:       scopeId,
	}).Unwrap()
}

func (s *RealmServiceImpl) InitializeRealmKeys(ctx context.Context) error {
	scope := middlewares.GetScope(ctx)

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realms := realmRepository.FindRealms(ctx, repos.RealmFilter{}).Unwrap()

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return err
	}

	keyCache := ioc.Get[cache.KeyCache](scope)
	for _, realm := range realms.Values() {
		decryptedPrivateKeyBytes, err := utils.DecryptSymmetric(realm.EncryptedPrivateKey, key)
		if err != nil {
			return err
		}

		privateKey, _, err := utils.ImportPrivateKey(decryptedPrivateKeyBytes)
		if err != nil {
			return err
		}

		keyCache.Set(realm.Id, privateKey)
	}

	return nil
}

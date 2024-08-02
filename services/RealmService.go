package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/cache"
	"holvit/config"
	"holvit/constants"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
)

type CreateRealmRequest struct {
	Name        string
	DisplayName string
}

type CreateRealmResponse struct {
	Id uuid.UUID
}

type RealmService interface {
	CreateRealm(ctx context.Context, request CreateRealmRequest) (*CreateRealmResponse, error)
	InitializeRealmKeys(ctx context.Context) error
}

type RealmServiceImpl struct {
}

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

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realmId, err := realmRepository.CreateRealm(ctx, &repositories.Realm{
		Name:                request.Name,
		DisplayName:         request.DisplayName,
		EncryptedPrivateKey: encryptedPrivateKeyBytes,
	})
	if err != nil {
		return nil, err
	}

	err = s.createOpenIdScope(ctx, realmId)
	if err != nil {
		return nil, err
	}

	err = s.createEmailScope(ctx, realmId)
	if err != nil {
		return nil, err
	}

	err = s.createProfileScope(ctx, realmId)
	if err != nil {
		return nil, err
	}

	keyCache := ioc.Get[cache.KeyCache](scope)
	keyCache.Set(realmId, privateKeyBytes)

	return &CreateRealmResponse{
		Id: realmId,
	}, nil
}

func (s *RealmServiceImpl) createProfileScope(ctx context.Context, realmId uuid.UUID) error {
	scope := middlewares.GetScope(ctx)

	scopeRepository := ioc.Get[repositories.ScopeRepository](scope)
	claimMapperRepository := ioc.Get[repositories.ClaimMapperRepository](scope)

	scopeId, err := scopeRepository.CreateScope(ctx, &repositories.Scope{
		RealmId:     realmId,
		Name:        "profile",
		DisplayName: "Profile",
		Description: "Access your name",
		SortIndex:   3,
	})
	if err != nil {
		return err
	}

	var claimId uuid.UUID
	claimId, err = claimMapperRepository.CreateClaimMapper(ctx, &repositories.ClaimMapper{
		BaseModel:   repositories.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Username",
		Description: "The username of the user",
		Type:        constants.ClaimMapperUserInfo,
		Details: repositories.UserInfoClaimMapperDetails{
			ClaimName: "preferred_username",
			Property:  constants.UserInfoPropertyUsername,
		},
	})
	if err != nil {
		return err
	}

	_, err = claimMapperRepository.AssociateClaimMapper(ctx, repositories.AssociateScopeClaimRequest{
		ClaimMapperId: claimId,
		ScopeId:       scopeId,
	})
	if err != nil {
		return err
	}

	return err
}

func (s *RealmServiceImpl) createEmailScope(ctx context.Context, realmId uuid.UUID) error {
	scope := middlewares.GetScope(ctx)

	scopeRepository := ioc.Get[repositories.ScopeRepository](scope)
	claimMapperRepository := ioc.Get[repositories.ClaimMapperRepository](scope)

	scopeId, err := scopeRepository.CreateScope(ctx, &repositories.Scope{
		RealmId:     realmId,
		Name:        "email",
		DisplayName: "Email",
		Description: "Access your email address",
		SortIndex:   2,
	})
	if err != nil {
		return err
	}

	var claimId uuid.UUID
	claimId, err = claimMapperRepository.CreateClaimMapper(ctx, &repositories.ClaimMapper{
		BaseModel:   repositories.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Email address",
		Description: "The primary email address of the user",
		Type:        constants.ClaimMapperUserInfo,
		Details: repositories.UserInfoClaimMapperDetails{
			ClaimName: "email",
			Property:  constants.UserInfoPropertyEmail,
		},
	})
	if err != nil {
		return err
	}

	_, err = claimMapperRepository.AssociateClaimMapper(ctx, repositories.AssociateScopeClaimRequest{
		ClaimMapperId: claimId,
		ScopeId:       scopeId,
	})
	if err != nil {
		return err
	}

	claimId, err = claimMapperRepository.CreateClaimMapper(ctx, &repositories.ClaimMapper{
		BaseModel:   repositories.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Email address verified",
		Description: "Whether the email address was verified or not",
		Type:        constants.ClaimMapperUserInfo,
		Details: repositories.UserInfoClaimMapperDetails{
			ClaimName: "email_verified",
			Property:  constants.UserInfoPropertyEmailVerified,
		},
	})
	if err != nil {
		return err
	}

	_, err = claimMapperRepository.AssociateClaimMapper(ctx, repositories.AssociateScopeClaimRequest{
		ClaimMapperId: claimId,
		ScopeId:       scopeId,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *RealmServiceImpl) createOpenIdScope(ctx context.Context, realmId uuid.UUID) error {
	scope := middlewares.GetScope(ctx)

	scopeRepository := ioc.Get[repositories.ScopeRepository](scope)
	claimMapperRepository := ioc.Get[repositories.ClaimMapperRepository](scope)

	scopeId, err := scopeRepository.CreateScope(ctx, &repositories.Scope{
		RealmId:     realmId,
		Name:        "openid",
		DisplayName: "OpenId Connect",
		Description: "Sign you in",
		SortIndex:   1,
	})
	if err != nil {
		return err
	}

	var claimId uuid.UUID
	claimId, err = claimMapperRepository.CreateClaimMapper(ctx, &repositories.ClaimMapper{
		BaseModel:   repositories.BaseModel{},
		RealmId:     realmId,
		DisplayName: "Subject Identifier",
		Description: "The id of the user",
		Type:        constants.ClaimMapperUserInfo,
		Details: repositories.UserInfoClaimMapperDetails{
			ClaimName: "sub",
			Property:  constants.UserInfoPropertyId,
		},
	})
	if err != nil {
		return err
	}

	_, err = claimMapperRepository.AssociateClaimMapper(ctx, repositories.AssociateScopeClaimRequest{
		ClaimMapperId: claimId,
		ScopeId:       scopeId,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *RealmServiceImpl) InitializeRealmKeys(ctx context.Context) error {
	scope := middlewares.GetScope(ctx)

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realms, _, err := realmRepository.FindRealms(ctx, repositories.RealmFilter{})
	if err != nil {
		return err
	}

	key, err := config.C.GetSymmetricEncryptionKey()
	if err != nil {
		return err
	}

	keyCache := ioc.Get[cache.KeyCache](scope)
	for _, realm := range realms {
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

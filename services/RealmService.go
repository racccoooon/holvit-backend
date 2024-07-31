package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/cache"
	"holvit/config"
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

	scopeRepository := ioc.Get[repositories.ScopeRepository](scope)
	_, err = scopeRepository.CreateScope(ctx, &repositories.Scope{
		RealmId:     realmId,
		Name:        "openid",
		DisplayName: "OpenId Connect",
		Description: "Sign you in",
	})
	if err != nil {
		return nil, err
	}

	_, err = scopeRepository.CreateScope(ctx, &repositories.Scope{
		RealmId:     realmId,
		Name:        "email",
		DisplayName: "Email",
		Description: "Access your email address",
	})
	if err != nil {
		return nil, err
	}

	_, err = scopeRepository.CreateScope(ctx, &repositories.Scope{
		RealmId:     realmId,
		Name:        "profile",
		DisplayName: "Profile",
		Description: "Access your name",
	})
	if err != nil {
		return nil, err
	}

	keyCache := ioc.Get[cache.KeyCache](scope)
	keyCache.Set(realmId, privateKeyBytes)

	return &CreateRealmResponse{
		Id: realmId,
	}, nil
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

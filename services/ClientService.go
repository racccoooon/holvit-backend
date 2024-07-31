package services

import (
	"context"
	"encoding/base64"
	"github.com/google/uuid"
	"holvit/config"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
)

type CreateClientRequest struct {
	RealmId     uuid.UUID
	DisplayName string
}

type CreateClientResponse struct {
	Id           uuid.UUID
	ClientId     string
	ClientSecret string
}

type ClientService interface {
	CreateClient(ctx context.Context, request CreateClientRequest) (*CreateClientResponse, error)
}

type ClientServiceImpl struct{}

func NewClientService() ClientService {
	return &ClientServiceImpl{}
}

func (c ClientServiceImpl) CreateClient(ctx context.Context, request CreateClientRequest) (*CreateClientResponse, error) {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repositories.ClientRepository](scope)

	clientId, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	clientIdString := clientId.String()

	clientSecretBytes, err := utils.GenerateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	hashAlgorithm := config.C.GetHashAlgorithm()
	hashedClientSecret, err := hashAlgorithm.Hash(clientSecretBytes)
	if err != nil {
		return nil, err
	}

	client := repositories.Client{
		RealmId:      request.RealmId,
		DisplayName:  request.DisplayName,
		ClientId:     clientIdString,
		ClientSecret: hashedClientSecret,
		RedirectUris: make([]string, 0),
	}

	clientDbId, err := clientRepository.CreateClient(ctx, &client)

	if err != nil {
		return nil, err
	}

	return &CreateClientResponse{
		Id:           clientDbId,
		ClientId:     clientIdString,
		ClientSecret: "secret_" + base64.StdEncoding.EncodeToString(clientSecretBytes),
	}, nil
}

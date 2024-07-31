package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/config"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
	"strings"
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

type AuthenticateClientRequest struct {
	ClientId     string
	ClientSecret string
}

type ClientService interface {
	CreateClient(ctx context.Context, request CreateClientRequest) (*CreateClientResponse, error)
	Authenticate(ctx context.Context, request AuthenticateClientRequest) (*repositories.Client, error)
}

type ClientServiceImpl struct{}

func NewClientService() ClientService {
	return &ClientServiceImpl{}
}

func (c *ClientServiceImpl) Authenticate(ctx context.Context, request AuthenticateClientRequest) (*repositories.Client, error) {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repositories.ClientRepository](scope)
	clients, count, err := clientRepository.FindClients(ctx, repositories.ClientFilter{
		ClientId: &request.ClientId,
	})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, httpErrors.NotFound().WithMessage("Client not found")
	}
	client := clients[0]

	requestClientSecret, _ := strings.CutPrefix(request.ClientSecret, "secret_")
	err = utils.CompareHash(requestClientSecret, client.ClientSecret)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *ClientServiceImpl) CreateClient(ctx context.Context, request CreateClientRequest) (*CreateClientResponse, error) {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repositories.ClientRepository](scope)

	clientId, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	clientIdString := clientId.String()

	clientSecret, err := utils.GenerateRandomString(32)
	if err != nil {
		return nil, err
	}

	hashAlgorithm := config.C.GetHashAlgorithm()
	hashedClientSecret, err := hashAlgorithm.Hash(clientSecret)
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
		ClientSecret: "secret_" + clientSecret,
	}, nil
}

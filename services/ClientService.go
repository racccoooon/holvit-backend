package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/config"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/utils"
	"strings"
)

type CreateClientRequest struct {
	RealmId     uuid.UUID
	ClientId    h.Optional[string]
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
	Authenticate(ctx context.Context, request AuthenticateClientRequest) (*repos.Client, error)
}

type ClientServiceImpl struct{}

func NewClientService() ClientService {
	return &ClientServiceImpl{}
}

func (c *ClientServiceImpl) Authenticate(ctx context.Context, request AuthenticateClientRequest) (*repos.Client, error) {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repos.ClientRepository](scope)
	client := clientRepository.FindClients(ctx, repos.ClientFilter{
		ClientId: h.Some(request.ClientId),
	}).First()

	requestClientSecret, _ := strings.CutPrefix(request.ClientSecret, "secret_")
	valid := utils.CompareHash(requestClientSecret, client.ClientSecret)
	if !valid {
		return nil, httpErrors.Unauthorized()
	}

	return &client, nil
}

func (c *ClientServiceImpl) CreateClient(ctx context.Context, request CreateClientRequest) (*CreateClientResponse, error) {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repos.ClientRepository](scope)

	clientId := request.ClientId.UnwrapOrElse(func() string {
		id, err := uuid.NewRandom()
		if err != nil {
			panic(err)
		}
		return id.String()
	})

	clientSecret := utils.GenerateRandomStringBase64(32)

	hashAlgorithm := config.C.GetHashAlgorithm()
	hashedClientSecret := hashAlgorithm.Hash(clientSecret)

	clientDbId := clientRepository.CreateClient(ctx, &repos.Client{
		RealmId:      request.RealmId,
		DisplayName:  request.DisplayName,
		ClientId:     clientId,
		ClientSecret: hashedClientSecret,
		RedirectUris: make([]string, 0),
	}).Unwrap()

	return &CreateClientResponse{
		Id:           clientDbId,
		ClientId:     clientId,
		ClientSecret: "secret_" + clientSecret,
	}, nil
}

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
	RealmId      uuid.UUID
	ClientId     h.Opt[string]
	DisplayName  string
	WithSecret   bool
	RedirectUrls []string
}

type CreateClientResponse struct {
	Id           uuid.UUID
	ClientId     string
	ClientSecret h.Opt[string]
}

type AuthenticateClientRequest struct {
	ClientId     string
	ClientSecret h.Opt[string]
}

type ClientService interface {
	CreateClient(ctx context.Context, request CreateClientRequest) CreateClientResponse
	Authenticate(ctx context.Context, request AuthenticateClientRequest) h.Result[repos.Client]
}

type clientServiceImpl struct{}

func NewClientService() ClientService {
	return &clientServiceImpl{}
}

// TODO: make sure all error responses in oidc comply with https://datatracker.ietf.org/doc/html/rfc6749#section-5.2

func (c *clientServiceImpl) Authenticate(ctx context.Context, request AuthenticateClientRequest) h.Result[repos.Client] {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repos.ClientRepository](scope)
	client := clientRepository.FindClients(ctx, repos.ClientFilter{
		ClientId: h.Some(request.ClientId),
	}).Single()

	if hashedSecret, ok := client.ClientSecret.Get(); ok {
		if providedSecret, ok := request.ClientSecret.Get(); ok {
			requestClientSecret, hasPrefix := strings.CutPrefix(providedSecret, "secret_")
			if !hasPrefix {
				return h.Err[repos.Client](httpErrors.Unauthorized().WithMessage("missing secret_ prefix"))
			}
			result := utils.ValidateHash(requestClientSecret, hashedSecret, config.C.GetHasher())
			if result.IsValid {
				if result.NeedsRehash {
					reHashed := config.C.GetHasher().Hash(requestClientSecret)
					clientRepository.UpdateClient(ctx, client.Id, repos.ClientUpdate{
						ClientSecret: h.Some(reHashed),
					}).Unwrap()
				}
				return h.Ok(client)
			}
			return h.Err[repos.Client](httpErrors.Unauthorized().WithMessage("wrong client secret"))
		}
		return h.Err[repos.Client](httpErrors.Unauthorized().WithMessage("client requires a secret"))
	} else {
		if request.ClientSecret.IsSome() {
			return h.Err[repos.Client](httpErrors.Unauthorized().WithMessage("secret provided for secret-less client, secret missing"))
		}
		return h.Ok(client)
	}
}

func (c *clientServiceImpl) CreateClient(ctx context.Context, request CreateClientRequest) CreateClientResponse {
	scope := middlewares.GetScope(ctx)

	clientRepository := ioc.Get[repos.ClientRepository](scope)

	clientId := request.ClientId.UnwrapOrElse(func() string {
		id, err := uuid.NewRandom()
		if err != nil {
			panic(err)
		}
		return id.String()
	})

	clientSecret := h.None[string]()
	if request.WithSecret {
		clientSecret = h.Some(utils.GenerateRandomStringBase64(33))
	}
	hashAlgorithm := config.C.GetHasher()
	hashedClientSecret := clientSecret.Map(hashAlgorithm.Hash)

	clientDbId := clientRepository.CreateClient(ctx, repos.Client{
		RealmId:      request.RealmId,
		DisplayName:  request.DisplayName,
		ClientId:     clientId,
		ClientSecret: hashedClientSecret,
		RedirectUris: request.RedirectUrls,
	}).Unwrap()

	return CreateClientResponse{
		Id:           clientDbId,
		ClientId:     clientId,
		ClientSecret: clientSecret.Map(func(secret string) string { return "secret_" + secret }),
	}
}

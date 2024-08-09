package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"holvit/config"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/utils"
	"time"
)

type GrantInfo struct {
	ClientId             uuid.UUID            `json:"clientId"`
	RealmId              uuid.UUID            `json:"realmId"`
	AuthorizationRequest AuthorizationRequest `json:"authorizationRequest"`
}

type CodeInfo struct {
	RealmId         uuid.UUID     `json:"realmId"`
	ClientId        string        `json:"clientId"`
	UserId          uuid.UUID     `json:"userId"`
	RedirectUri     string        `json:"redirectUri"`
	GrantedScopes   []string      `json:"grantedScopes"`
	GrantedScopeIds []uuid.UUID   `json:"grantedScopeIds"`
	PKCEChallenge   h.Opt[string] `json:"pkceChallenge"`
}

type LoginInfo struct {
	NextStep                            string    `json:"nextStep"`
	RealmId                             uuid.UUID `json:"realmId"`
	UserId                              uuid.UUID `json:"userId"`
	DeviceId                            string    `json:"deviceId"`
	RememberMe                          bool      `json:"rememberMe"`
	EncryptedTotpOnboardingSecretBase64 string    `json:"totpSecret"`
	OriginalUrl                         string    `json:"originalUrl"`
}

type TokenService interface {
	StoreGrantInfo(ctx context.Context, info GrantInfo) string
	RetrieveGrantInfo(ctx context.Context, token string) h.Opt[GrantInfo]

	StoreOidcCode(ctx context.Context, info CodeInfo) string
	RetrieveOidcCode(ctx context.Context, token string) h.Opt[CodeInfo]

	StoreLoginCode(ctx context.Context, info LoginInfo) string
	OverwriteLoginCode(ctx context.Context, token string, info LoginInfo) h.Result[h.Unit]
	PeekLoginCode(ctx context.Context, token string) h.Opt[LoginInfo]
	RetrieveLoginCode(ctx context.Context, token string) h.Opt[LoginInfo]
}

type TokenServiceImpl struct{}

func (s *TokenServiceImpl) OverwriteLoginCode(ctx context.Context, token string, info LoginInfo) h.Result[h.Unit] {
	found := s.overwriteInfo(ctx, info, "loginCode", token, time.Minute*30) // TODO config
	if !found {
		return h.UErr(httpErrors.NotFound().WithMessage(fmt.Sprintf("login code %s not found", token)))
	}
	return h.UOk()
}

func (s *TokenServiceImpl) StoreLoginCode(ctx context.Context, info LoginInfo) string {
	return s.storeInfo(ctx, info, "loginCode", time.Minute*30) // TODO config
}

func (s *TokenServiceImpl) PeekLoginCode(ctx context.Context, token string) h.Opt[LoginInfo] {
	var result LoginInfo
	found := s.peekInfo(ctx, "loginCode", token, &result)
	return h.SomeIf(found, result)
}

func (s *TokenServiceImpl) RetrieveLoginCode(ctx context.Context, token string) h.Opt[LoginInfo] {
	var result LoginInfo
	found := s.retrieveInfo(ctx, "loginCode", token, &result)
	return h.SomeIf(found, result)
}

func (s *TokenServiceImpl) StoreOidcCode(ctx context.Context, info CodeInfo) string {
	return s.storeInfo(ctx, info, "oidcCode", time.Second*30)
}

func (s *TokenServiceImpl) RetrieveOidcCode(ctx context.Context, token string) h.Opt[CodeInfo] {
	var result CodeInfo
	found := s.retrieveInfo(ctx, "oidcCode", token, &result)
	return h.SomeIf(found, result)
}

func (s *TokenServiceImpl) StoreGrantInfo(ctx context.Context, info GrantInfo) string {
	return s.storeInfo(ctx, info, "grantInfo", time.Minute*5)
}

func (s *TokenServiceImpl) RetrieveGrantInfo(ctx context.Context, token string) h.Opt[GrantInfo] {
	var result GrantInfo
	found := s.retrieveInfo(ctx, "grantInfo", token, &result)
	return h.SomeIf(found, result)
}

func (s *TokenServiceImpl) storeInfo(ctx context.Context, info interface{}, name string, expiration time.Duration) string {
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)
	tokenBytes, err := utils.GenerateRandomBytes(32)
	if err != nil {
		panic(err)
	}

	token := base64.StdEncoding.EncodeToString(tokenBytes)
	logging.Logger.Debugf("storing redis: %s:%s", name, token)

	dataBytes, err := json.Marshal(info)
	if err != nil {
		panic(err)
	}

	data := string(dataBytes)

	if config.C.IsDevelopment() {
		expiration = time.Hour * 24
	}
	if err := redisClient.Set(ctx, name+":"+token, data, expiration).Err(); err != nil {
		panic(err)
	}

	return token
}

func (s *TokenServiceImpl) overwriteInfo(ctx context.Context, info interface{}, name string, token string, expiration time.Duration) bool {
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)

	logging.Logger.Debugf("overwriting redis: %s", token)

	dataBytes, err := json.Marshal(info)
	if err != nil {
		panic(err)
	}

	data := string(dataBytes)

	if config.C.IsDevelopment() {
		expiration = time.Hour * 24
	}
	//TODO: check if it was in redis to begin with
	if err := redisClient.Set(ctx, name+":"+token, data, expiration).Err(); err != nil {
		panic(err)
	}

	return true
}

func (s *TokenServiceImpl) retrieveInfo(ctx context.Context, name string, token string, info interface{}) bool {
	logging.Logger.Debugf("retrieving redis: %s:%s", name, token)
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)
	val, err := redisClient.GetDel(ctx, name+":"+token).Result()
	if errors.Is(err, redis.Nil) {
		return false
	}
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(val), info)
	if err != nil {
		panic(err)
	}

	return true
}

func (s *TokenServiceImpl) peekInfo(ctx context.Context, name string, token string, info interface{}) bool {
	logging.Logger.Debugf("peeking redis: %s:%s", name, token)
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)
	val, err := redisClient.Get(ctx, name+":"+token).Result()
	if errors.Is(err, redis.Nil) {
		return false
	}
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(val), info)
	if err != nil {
		panic(err)
	}

	return true
}

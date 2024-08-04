package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"holvit/config"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/utils"
	"time"
)

type GrantInfo struct {
	ClientId             uuid.UUID            `json:"client_id"`
	RealmId              uuid.UUID            `json:"realm_id"`
	AuthorizationRequest AuthorizationRequest `json:"authorization_request"`
}

type CodeInfo struct {
	RealmId         uuid.UUID   `json:"realm_id"`
	ClientId        string      `json:"client_id"`
	UserId          uuid.UUID   `json:"user_id"`
	RedirectUri     string      `json:"redirect_uri"`
	GrantedScopes   []string    `json:"granted_scopes"`
	GrantedScopeIds []uuid.UUID `json:"granted_scope_ids"`
}

type LoginInfo struct {
	NextStep                            string               `json:"next_step"`
	RealmId                             uuid.UUID            `json:"realm_id"`
	UserId                              uuid.UUID            `json:"user_id"`
	DeviceId                            string               `json:"device_id"`
	RememberMe                          bool                 `json:"remember_me"`
	Request                             AuthorizationRequest `json:"request"`
	EncryptedTotpOnboardingSecretBase64 string               `json:"totp_secret"`
}

type TokenService interface {
	StoreGrantInfo(ctx context.Context, info GrantInfo) (string, error)
	RetrieveGrantInfo(ctx context.Context, token string) (*GrantInfo, error)

	StoreOidcCode(ctx context.Context, info CodeInfo) (string, error)
	RetrieveOidcCode(ctx context.Context, token string) (*CodeInfo, error)

	StoreLoginCode(ctx context.Context, info LoginInfo) (string, error)
	OverwriteLoginCode(ctx context.Context, token string, info LoginInfo) error
	PeekLoginCode(ctx context.Context, token string) (*LoginInfo, error)
	RetrieveLoginCode(ctx context.Context, token string) (*LoginInfo, error)
}

type TokenServiceImpl struct{}

func (s *TokenServiceImpl) OverwriteLoginCode(ctx context.Context, token string, info LoginInfo) error {
	err := s.overwriteInfo(ctx, info, token, time.Minute*30) // TODO config
	if err != nil {
		return err
	}
	return nil
}

func (s *TokenServiceImpl) StoreLoginCode(ctx context.Context, info LoginInfo) (string, error) {
	token, err := s.storeInfo(ctx, info, "loginCode", time.Minute*30) // TODO config
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *TokenServiceImpl) PeekLoginCode(ctx context.Context, token string) (*LoginInfo, error) {
	var result LoginInfo
	err := s.peekInfo(ctx, "loginCode", token, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *TokenServiceImpl) RetrieveLoginCode(ctx context.Context, token string) (*LoginInfo, error) {
	var result LoginInfo
	err := s.retrieveInfo(ctx, "loginCode", token, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *TokenServiceImpl) StoreOidcCode(ctx context.Context, info CodeInfo) (string, error) {
	token, err := s.storeInfo(ctx, info, "oidcCode", time.Second*30)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *TokenServiceImpl) RetrieveOidcCode(ctx context.Context, token string) (*CodeInfo, error) {
	var result CodeInfo
	err := s.retrieveInfo(ctx, "oidcCode", token, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *TokenServiceImpl) StoreGrantInfo(ctx context.Context, info GrantInfo) (string, error) {
	token, err := s.storeInfo(ctx, info, "grantInfo", time.Minute*5)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *TokenServiceImpl) RetrieveGrantInfo(ctx context.Context, token string) (*GrantInfo, error) {
	var result GrantInfo
	err := s.retrieveInfo(ctx, "grantInfo", token, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *TokenServiceImpl) storeInfo(ctx context.Context, info interface{}, name string, expiration time.Duration) (string, error) {
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)
	tokenBytes, err := utils.GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}

	token := base64.StdEncoding.EncodeToString(tokenBytes)
	logging.Logger.Debugf("storing redis: %s:%s", name, token)

	dataBytes, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	data := string(dataBytes)

	if config.C.IsDevelopment() {
		expiration = time.Hour * 24
	}
	if err := redisClient.Set(ctx, name+":"+token, data, expiration).Err(); err != nil {
		return "", err
	}

	return token, nil
}

func (s *TokenServiceImpl) overwriteInfo(ctx context.Context, info interface{}, token string, expiration time.Duration) error {
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)

	logging.Logger.Debugf("overwriting redis: %s", token)

	dataBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	data := string(dataBytes)

	if config.C.IsDevelopment() {
		expiration = time.Hour * 24
	}
	//TODO: check if it was in redis to begin with
	if err := redisClient.Set(ctx, token, data, expiration).Err(); err != nil {
		return err
	}

	return nil
}

func (s *TokenServiceImpl) retrieveInfo(ctx context.Context, name string, token string, info interface{}) error {
	logging.Logger.Debugf("retrieving redis: %s:%s", name, token)
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)
	val, err := redisClient.GetDel(ctx, name+":"+token).Result()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(val), info)
	if err != nil {
		return err
	}

	return nil
}

func (s *TokenServiceImpl) peekInfo(ctx context.Context, name string, token string, info interface{}) error {
	logging.Logger.Debugf("peeking redis: %s:%s", name, token)
	scope := middlewares.GetScope(ctx)
	redisClient := ioc.Get[*redis.Client](scope)
	val, err := redisClient.Get(ctx, name+":"+token).Result()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(val), info)
	if err != nil {
		return err
	}

	return nil
}

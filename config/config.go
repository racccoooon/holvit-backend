package config

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"holvit/constants"
	"holvit/utils"
	"time"
)

type HolvitConfig struct {
	Environment string

	BaseUrl string

	Development struct {
		AuthFrontendUrl  string
		AdminFrontendUrl string
	}

	StaticRoot string

	Secret string

	MasterRealmName        string
	MasterRealmDisplayName string

	AdminUserName        string
	InitialAdminPassword string

	HashAlgorithm    string
	BCryptSettings   utils.BcryptHashSettings
	SCryptSettings   utils.ScryptHashSettings
	Argon2idSettings utils.Argon2idHashSettings

	Totp struct {
		Period uint
		Skew   uint
	}

	Server struct {
		Host            string
		Port            int
		WriteTimeout    time.Duration
		ReadTimeout     time.Duration
		ShutdownTimeout time.Duration
		MaxReadBytes    int64
	}

	UseMailServer bool
	MailServer    struct {
		From          string
		Host          string
		Port          int
		User          string
		Password      string
		StartTls      bool
		AllowInsecure bool
	}

	Database struct {
		Host     string
		Port     int
		Username string
		Password string
		Database string
		SslMode  string
	}

	Redis struct {
		Host     string
		Port     int
		Password string
		Db       int
		Protocol int
	}

	Crons struct {
		JobScheduler   string
		SessionCleanup string
	}
}

const (
	Production = "Production"
	Staging    = "Staging"
)

func (c *HolvitConfig) IsProduction() bool {
	return c.Environment == Production
}

func (c *HolvitConfig) IsStaging() bool {
	return c.Environment == Staging
}

func (c *HolvitConfig) IsDevelopment() bool {
	return !(c.IsStaging() || c.IsProduction())
}

var hasher utils.Hasher

func (c *HolvitConfig) GetHasher() utils.Hasher {
	return hasher
}

func (c *HolvitConfig) getHashSettings() utils.HashSettings {
	switch C.HashAlgorithm {
	case constants.HashAlgorithmBCrypt:
		return &C.BCryptSettings
	case constants.HashAlgorithmSCrypt:
		return &C.SCryptSettings
	case constants.HashAlgorithmArgon2id:
		return &C.Argon2idSettings
	default:
		panic(fmt.Errorf("hash algorithm '%s' is not supported", c.HashAlgorithm))
	}
}

func (c *HolvitConfig) GetSymmetricEncryptionKey() []byte {
	return utils.GenerateSymmetricKeyFromText(C.Secret)
}

var C HolvitConfig
var configFilePath string

func Init() {
	readFlags()
	setDefaultConfigValues()
	readConfigValues()
	validateConfig()
	hasher = C.getHashSettings().MakeHasher()
}

func readFlags() {
	flag.StringVar(&configFilePath, "config", "./config.yml", "config file path")
	flag.StringVar(&configFilePath, "c", "./config.yml", "config file path (shorthand)")
}

func setDefaultConfigValues() {
	C.Environment = Production
	C.BaseUrl = "http://localhost:8080"

	C.Development.AuthFrontendUrl = "http://localhost:5173/"
	C.Development.AdminFrontendUrl = "http://localhost:5174/"

	C.StaticRoot = "/static/"

	C.HashAlgorithm = constants.HashAlgorithmArgon2id
	C.BCryptSettings.Cost = 12
	C.SCryptSettings.R = 8
	C.SCryptSettings.LogN = 32768
	C.SCryptSettings.Parallelism = 1
	C.SCryptSettings.SaltLength = 32
	C.SCryptSettings.OutputLength = 32
	C.Argon2idSettings.OpsCost = 3
	C.Argon2idSettings.Parallelism = 1
	C.Argon2idSettings.MemoryCost = 64 * 1024
	C.Argon2idSettings.SaltLength = 16
	C.Argon2idSettings.OutputLength = 32

	C.Totp.Period = 30
	C.Totp.Skew = 1

	C.MasterRealmName = "admin"
	C.MasterRealmDisplayName = "Admin Realm"

	C.AdminUserName = "admin"

	C.Server.Host = "0.0.0.0"
	C.Server.Port = 8080

	C.Server.ReadTimeout = 15 * time.Second
	C.Server.WriteTimeout = 15 * time.Second

	C.Server.ShutdownTimeout = 15 * time.Second

	C.Server.MaxReadBytes = 1048576

	C.Database.Host = "localhost"
	C.Database.Port = 5432
	C.Database.Database = "holvit"
	C.Database.SslMode = "disable"

	C.Redis.Host = "localhost"
	C.Redis.Port = 6379
	C.Redis.Password = ""
	C.Redis.Db = 0
	C.Redis.Protocol = 3

	C.Crons.JobScheduler = "* * * * *"
}

func readConfigValues() {
	v := viper.NewWithOptions(viper.KeyDelimiter("_"))

	v.SetEnvPrefix("HOLVIT")
	v.AutomaticEnv()

	v.SetConfigFile(configFilePath)

	err := v.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		err = nil
	}
	if err != nil {
		panic(err)
	}

	err = v.Unmarshal(&C)
	if err != nil {
		panic(err)
	}
}

func validateConfig() {
	//TODO
}

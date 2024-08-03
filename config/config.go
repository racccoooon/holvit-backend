package config

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"holvit/constants"
	"holvit/utils"
	"time"
)

type HolvitConfig struct {
	Environment string

	Secret string

	MasterRealmName        string
	MasterRealmDisplayName string

	AdminUserName        string
	InitialAdminPassword string

	HashAlgorithm  string
	BCryptSettings utils.BCryptHashAlgorithm

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
		JobScheduler string
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

func (c *HolvitConfig) GetHashAlgorithm() utils.HashAlgorithm {
	switch C.HashAlgorithm {
	case constants.HashAlgorithmBCrypt:
		return &C.BCryptSettings
	default:
		panic(fmt.Sprint("Hash algorithm '%s' is not supported", c.HashAlgorithm))
	}
}

func (c *HolvitConfig) GetSymmetricEncryptionKey() ([]byte, error) {
	return utils.GenerateSymmetricKeyFromText(C.Secret)
}

var C HolvitConfig
var configFilePath string

func Init() {
	readFlags()
	setDefaultConfigValues()
	readConfigValues()
	validateConfig()
}

func readFlags() {
	flag.StringVar(&configFilePath, "config", "./config.yml", "config file path")
	flag.StringVar(&configFilePath, "c", "./config.yml", "config file path (shorthand)")
}

func setDefaultConfigValues() {
	C.Environment = Production

	C.HashAlgorithm = constants.HashAlgorithmBCrypt
	C.BCryptSettings.Cost = bcrypt.DefaultCost

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

	v.Unmarshal(&C)
}

func validateConfig() {
	//TODO
}

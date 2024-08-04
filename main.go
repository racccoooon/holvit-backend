package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/huandu/go-sqlbuilder"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"holvit/cache"
	"holvit/config"
	"holvit/crons"
	"holvit/database"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/requestContext"
	"holvit/server"
	"holvit/services"
	"holvit/utils"
	"os"
)

func main() {
	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	config.Init()
	logging.Init()

	logging.Logger.Info("Application starting...")
	logging.Logger.Infof("Using environment '%s'", config.C.Environment)

	database.Migrate()

	ioc.RootScope = configureServices()

	initialize(ioc.RootScope)
	server.ServeApi(ioc.RootScope)

	logging.Logger.Info("Application shutting down...")
	os.Exit(0)
}

func initialize(dp *ioc.DependencyProvider) {
	scope := dp.NewScope()
	defer scope.Close()

	ctx := middlewares.ContextWithNewScope(context.Background(), scope)

	realmRepository := ioc.Get[repositories.RealmRepository](scope)

	_, realmCount, err := realmRepository.FindRealms(ctx, repositories.RealmFilter{
		BaseFilter: repositories.BaseFilter{},
	})
	if err != nil {
		logging.Logger.Fatal(err)
	}

	if realmCount == 0 {
		seedData(ctx)
	}

	initializeApplicationData(ctx)
}

func initializeApplicationData(ctx context.Context) {
	scope := middlewares.GetScope(ctx)

	logging.Logger.Info("Initializing application...")

	realmService := ioc.Get[services.RealmService](scope)
	err := realmService.InitializeRealmKeys(ctx)
	if err != nil {
		logging.Logger.Fatal(err)
	}
}

func seedData(ctx context.Context) {
	scope := middlewares.GetScope(ctx)

	logging.Logger.Info("Seeding data...")

	realmService := ioc.Get[services.RealmService](scope)
	masterRealm, err := realmService.CreateRealm(ctx, services.CreateRealmRequest{
		Name:        config.C.MasterRealmName,
		DisplayName: config.C.MasterRealmDisplayName,
	})
	if err != nil {
		logging.Logger.Fatal(err)
	}

	clientService := ioc.Get[services.ClientService](scope)
	clientResponse, err := clientService.CreateClient(ctx, services.CreateClientRequest{
		RealmId:     masterRealm.Id,
		DisplayName: "Holvit Admin",
	})
	if err != nil {
		logging.Logger.Fatal(err)
	}
	logging.Logger.Infof("admin client id=%s secret=%s", clientResponse.ClientId, clientResponse.ClientSecret)

	userService := ioc.Get[services.UserService](scope)
	adminUser, err := userService.CreateUser(ctx, services.CreateUserRequest{
		RealmId:  masterRealm.Id,
		Username: &config.C.AdminUserName,
		Email:    nil,
	})
	if err != nil {
		logging.Logger.Fatal(err)
	}

	err = userService.SetPassword(ctx, services.SetPasswordRequest{
		UserId:    adminUser.Id,
		Password:  config.C.InitialAdminPassword,
		Temporary: true,
	})
	if err != nil {
		logging.Logger.Fatal(err)
	}
}

func configureServices() *ioc.DependencyProvider {
	builder := ioc.NewDependencyProviderBuilder()
	db := database.ConnectToDatabase()
	c := cron.New()

	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) *sql.DB {
		return db
	})
	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) utils.ClockService {
		return utils.NewClockService()
	})
	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) *cron.Cron {
		return c
	})

	ioc.AddScoped(builder, func(dp *ioc.DependencyProvider) requestContext.RequestContextService {
		return requestContext.NewRequestContextService(dp)
	})
	ioc.AddCloseHandler[requestContext.RequestContextService](builder, func(rcs requestContext.RequestContextService) error {
		return rcs.Close()
	})
	ioc.AddScoped(builder, func(dp *ioc.DependencyProvider) services.CurrentUserService {
		return services.NewCurrentUserService()
	})

	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.RealmRepository {
		return repositories.NewRealmRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.UserRepository {
		return repositories.NewUserRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.CredentialRepository {
		return repositories.NewCredentialRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.ClientRepository {
		return repositories.NewClientRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.ScopeRepository {
		return repositories.NewScopeReposiroty()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.RefreshTokenRepository {
		return repositories.NewRefreshTokenRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.ClaimMapperRepository {
		return repositories.NewClaimMapperRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.UserDeviceRepository {
		return repositories.NewUserDeviceRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repositories.QueuedJobRepository {
		return repositories.NewQueuedJobRepository()
	})

	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.UserService {
		return services.NewUserService()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.RealmService {
		return services.NewRealmService()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.ClientService {
		return services.NewClientService()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.RefreshTokenService {
		return services.NewRefreshTokenService()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.ClaimsService {
		return services.NewClaimsService()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.SessionService {
		return services.NewSessionService()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.DeviceService {
		return services.NewDeviceService()
	})

	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.OidcService {
		return services.NewOidcService()
	})

	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) services.JobService {
		return services.NewJobService(c)
	})

	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) cache.KeyCache {
		return cache.NewInMemoryKeyCache()
	})

	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) services.TokenService {
		return &services.TokenServiceImpl{}
	})

	ioc.Add(builder, func(dp *ioc.DependencyProvider) *redis.Client {
		return redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", config.C.Redis.Host, config.C.Redis.Port),
			Password: config.C.Redis.Password,
			DB:       config.C.Redis.Db,
			Protocol: config.C.Redis.Protocol,
		})
	})

	// configure crons
	c.AddFunc(config.C.Crons.SessionCleanup, crons.SessionCleanup)

	c.Start()

	return builder.Build()
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jaswdr/faker/v2"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"holvit/cache"
	"holvit/config"
	"holvit/constants"
	"holvit/crons"
	"holvit/database"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
	"holvit/routes"
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
	server.Serve(ioc.RootScope)

	logging.Logger.Info("Application shutting down...")
	os.Exit(0)
}

func initialize(dp *ioc.DependencyProvider) {
	requestContext.RunWithScope(dp, context.Background(), func(ctx context.Context) {
		scope := middlewares.GetScope(ctx)
		realmRepository := ioc.Get[repos.RealmRepository](scope)

		realmsResult := realmRepository.FindRealms(ctx, repos.RealmFilter{
			BaseFilter: repos.BaseFilter{},
		})

		if !realmsResult.Any() {
			seedData(ctx)
			seedDemoData(ctx)
		}

		initializeApplicationData(ctx)
	})
}

func initializeApplicationData(ctx context.Context) {
	scope := middlewares.GetScope(ctx)

	logging.Logger.Info("Initializing application...")

	realmService := ioc.Get[services.RealmService](scope)
	realmService.InitializeRealmKeys(ctx)
}

func seedData(ctx context.Context) {
	scope := middlewares.GetScope(ctx)

	logging.Logger.Info("Seeding data...")

	realmService := ioc.Get[services.RealmService](scope)
	masterRealm := realmService.CreateRealm(ctx, services.CreateRealmRequest{
		Name:        constants.MasterRealmName,
		DisplayName: "Admin Realm",
	})

	clientService := ioc.Get[services.ClientService](scope)
	adminClient := clientService.CreateClient(ctx, services.CreateClientRequest{
		RealmId:      masterRealm.Id,
		ClientId:     h.Some("holvit_admin"),
		DisplayName:  "Holvit Admin",
		WithSecret:   false,
		RedirectUrls: []string{routes.AdminFrontend.Url()},
	})

	logging.Logger.Infof("admin client id=%s secret=%s", adminClient.ClientId, adminClient.ClientSecret)

	userService := ioc.Get[services.UserService](scope)
	adminUserId := userService.CreateUser(ctx, services.CreateUserRequest{
		RealmId:  masterRealm.Id,
		Username: config.C.AdminUserName,
	}).Unwrap()

	roleRepository := ioc.Get[repos.RoleRepository](scope)
	superUserRole := roleRepository.FindRoles(ctx, repos.RoleFilter{
		RealmId: masterRealm.Id,
		Name:    h.Some(constants.SuperUserRoleName),
	}).Single()

	roleService := ioc.Get[services.RoleService](scope)
	roleService.AssignRolesToUser(ctx, services.AssignRolesToUserRequest{
		RealmId: masterRealm.Id,
		UserId:  adminUserId,
		RoleIds: []uuid.UUID{superUserRole.Id},
	})

	userService.SetPassword(ctx, services.SetPasswordRequest{
		UserId:    adminUserId,
		Password:  config.C.InitialAdminPassword,
		Temporary: true,
	}, services.DangerousNoAuthStrategy{})
}

func seedDemoData(ctx context.Context) {
	if !config.C.IsDevelopment() {
		return
	}
	fake := faker.New()

	scope := middlewares.GetScope(ctx)

	logging.Logger.Info("Seeding demo data...")

	realmService := ioc.Get[services.RealmService](scope)
	realm := realmService.CreateRealm(ctx, services.CreateRealmRequest{
		Name:        "demo",
		DisplayName: "Demo Realm",
	})

	for i := 0; i < 5; i++ {
		clientService := ioc.Get[services.ClientService](scope)
		_ = clientService.CreateClient(ctx, services.CreateClientRequest{
			RealmId:      realm.Id,
			ClientId:     h.Some(fake.Internet().Slug()),
			DisplayName:  fake.App().Name(),
			WithSecret:   false,
			RedirectUrls: []string{fake.Internet().Domain()},
		})
	}

	for i := 0; i < 10; i++ {
		userService := ioc.Get[services.UserService](scope)
		userId := userService.CreateUser(ctx, services.CreateUserRequest{
			RealmId:  realm.Id,
			Username: fake.Gamer().Tag(),
			Email:    h.Some(fake.Internet().Email()),
		}).Unwrap()

		userService.SetPassword(ctx, services.SetPasswordRequest{
			UserId:    userId,
			Password:  config.C.InitialAdminPassword,
			Temporary: false,
		}, services.DangerousNoAuthStrategy{})
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

	ioc.AddSingleton(builder, func(dp *ioc.DependencyProvider) services.FrontendService { return services.NewFrontendService() })

	ioc.AddScoped(builder, func(dp *ioc.DependencyProvider) requestContext.RequestContextService {
		return requestContext.NewRequestContextService(dp)
	})
	ioc.AddCloseHandler[requestContext.RequestContextService](builder, func(rcs requestContext.RequestContextService) error {
		return rcs.Close()
	})
	ioc.AddScoped(builder, func(dp *ioc.DependencyProvider) services.CurrentSessionService {
		return services.NewCurrentSessionService()
	})

	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.RealmRepository {
		return repos.NewRealmRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.UserRepository {
		return repos.NewUserRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.CredentialRepository {
		return repos.NewCredentialRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.ClientRepository {
		return repos.NewClientRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.ScopeRepository {
		return repos.NewScopeRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.RoleRepository {
		return repos.NewRoleRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.RefreshTokenRepository {
		return repos.NewRefreshTokenRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.ClaimMapperRepository {
		return repos.NewClaimMapperRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.UserDeviceRepository {
		return repos.NewUserDeviceRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.QueuedJobRepository {
		return repos.NewQueuedJobRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.SessionRepository {
		return repos.NewSessionRepository()
	})
	ioc.Add(builder, func(dp *ioc.DependencyProvider) repos.UserRoleRepository {
		return repos.NewUserRoleRepository()
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
	ioc.Add(builder, func(dp *ioc.DependencyProvider) services.RoleService {
		return services.NewRoleService()
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
		return services.NewTokenService()
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

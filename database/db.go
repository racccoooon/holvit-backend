package database

import (
	"database/sql"
	"embed"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
	"holvit/config"
	"holvit/logging"
)

//go:embed migrations/*
var dbMigrations embed.FS

func ConnectToDatabase() *sql.DB {
	logging.Logger.Infof("Connecting to database %s via %s:%d",
		config.C.Database.Database,
		config.C.Database.Host,
		config.C.Database.Port)

	connection, err := connectToDatabase(
		config.C.Database.Host,
		config.C.Database.Port,
		config.C.Database.Username,
		config.C.Database.Password,
		config.C.Database.Database,
		config.C.Database.SslMode)
	if err != nil {
		logging.Logger.Fatalf("Failed to connect to the database: %v", err)
	}

	return connection
}

func connectToDatabase(host string,
	port int,
	user string,
	password string,
	database string,
	sslMode string) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		host,
		port,
		database,
		user,
		password,
		sslMode)

	return sql.Open("postgres", connectionString)
}

func Migrate() {
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	db := ConnectToDatabase()
	defer db.Close()

	logging.Logger.Info("Applying migrations...")

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		panic(err)
	}

	logging.Logger.Infof("Applied %d migrations", n)
}

package api

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Mode      string
	ApiPort   string
	LogConfig struct {
		Enabled   bool
		QueueName string
	}
	MainDatabase struct {
		Host         string
		Port         string
		User         string
		Password     string
		DatabaseName string
		SSLMode      string
	}
	JWTConfig struct {
		Secret            string
		Expiration        int // in minutes
		RefreshExpiration int // in days
	}
}

var config AppConfig

func InitConfig(envfile string) {
	err := godotenv.Load(envfile)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error loading %s file: %s", envfile, err))
	} else {
		log.Println("Loaded .env file successfully")
	}
	config = AppConfig{
		Mode:    getEnvOrPanic("RUN_MODE"),
		ApiPort: getEnvOrPanic("API_PORT"),
		MainDatabase: struct {
			Host         string
			Port         string
			User         string
			Password     string
			DatabaseName string
			SSLMode      string
		}{
			Host:         getEnvOrPanic("DB_HOSTNAME"),
			Port:         getEnvOrPanic("DB_PORT"),
			User:         getEnvOrPanic("DB_USERNAME"),
			Password:     getEnvOrPanic("DB_PASSWORD"),
			DatabaseName: getEnvOrPanic("DB_NAME"),
			SSLMode:      getEnvOrPanic("DB_SSL_MODE"),
		},
		LogConfig: struct {
			Enabled   bool
			QueueName string
		}{
			Enabled:   true,
			QueueName: "logs",
		},
		JWTConfig: struct {
			Secret            string
			Expiration        int
			RefreshExpiration int
		}{
			Secret:            getEnvOrPanic("JWT_SECRET"),
			Expiration:        getIntEnvOrPanic("JWT_EXPIRATION_MINUTES"),
			RefreshExpiration: getIntEnvOrPanic("JWT_REFRESH_EXPIRATION_DAYS"),
		},
	}

	DB = connectToPostgres(config.MainDatabase.Host, config.MainDatabase.User, config.MainDatabase.Password, config.MainDatabase.DatabaseName, config.MainDatabase.Port, config.MainDatabase.SSLMode)
	Logger = initLogger()
}

func GetConfig() AppConfig {
	return config
}

func getEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s must be set", key)
	}
	return value
}

func GetEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getIntEnvOrPanic(key string) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Fatalf("%s must be an integer", key)
	}
	return value
}

func connectToPostgres(host string, username string, password string, dbname string, port string, ssl string) *gorm.DB {
	var err error
	var db *gorm.DB
	var conn *sql.DB

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, username, password, dbname, port, ssl)
	if db, err = gorm.Open(postgres.Open(dsn),
		&gorm.Config{
			Logger: logger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold: 0,
					LogLevel:      logger.Error,
				},
			),
			FullSaveAssociations: true,
			CreateBatchSize:      1000,
			TranslateError:       true,
			NowFunc: func() time.Time {
				//loc, err := time.LoadLocation("Europe/Paris")
				//if err != nil {
				//	panic(err)
				//}
				// No idea why but it's not working without adding 1 hour
				//return time.Now().In(loc).Add(time.Hour * 1)
				//return time.Now().Add(time.Hour * 1)
				return time.Now()
			},
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:         "",
				SingularTable:       true,
				NameReplacer:        nil,
				NoLowerCase:         false,
				IdentifierMaxLength: 0,
			}}); err != nil {
		panic(err)
	}
	if conn, err = db.DB(); err != nil {
		panic(err)
	}
	conn.SetMaxIdleConns(10)
	conn.SetMaxOpenConns(10)
	conn.SetConnMaxLifetime(time.Hour)
	return db
}

func initLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
}

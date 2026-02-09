package main

import (
	"api"
	"api/internal/api/handler/endpoints"
	"api/internal/api/models"
	"api/internal/api/service"
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
)

func main() {
	api.InitConfig(".env")
	gin.SetMode(gin.ReleaseMode)

	if api.GetConfig().Mode == "dev" {
		if err := api.DB.AutoMigrate(
			&models.User{},
			&models.Job{},
			&models.Node{},
			&models.Port{},
			&models.MetadataDatabase{},
			&models.MetadataSftp{},
			&models.MetadataEmail{},
			&models.JobUserAccess{},
			// Trigger system models
			&models.Trigger{},
			&models.TriggerRule{},
			&models.TriggerJob{},
			&models.TriggerExecution{},
		); err != nil {
			api.Logger.Fatal().Err(err).Msg("Failed to migrate database")
		}
		api.Logger.Info().Msg("Database migrated successfully")
		gin.SetMode(gin.DebugMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	router, err := graceful.Default(graceful.WithAddr(api.GetConfig().ApiPort))
	if err != nil {
		panic(err)
	}
	defer stop()
	defer router.Close()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	initAPI(router)

	// Start the trigger polling service
	pollerService := service.NewTriggerPollerService(10) // Max 10 concurrent workers
	pollerService.Start()
	defer pollerService.Stop()

	api.Logger.Debug().Msgf("Starting CORE API on port %s", api.GetConfig().ApiPort)
	if err = router.RunWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		api.Logger.Fatal().Msg(err.Error())
		panic(err)
	}

}

func initAPI(router *graceful.Graceful) {
	endpoints.AuthHandler(router)
	endpoints.DbMetadataHandler(router)
	endpoints.DbNodeHandler(router)
	endpoints.JobHandler(router)
	endpoints.SqlHandler(router)
	endpoints.TriggerHandler(router)
}

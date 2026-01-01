package app

import (
	"context"
	"time"

	"github.com/creafly/logger"
	"github.com/creafly/storage/internal/config"
	"github.com/creafly/storage/internal/domain/repository"
	"github.com/creafly/storage/internal/domain/service"
	"github.com/creafly/storage/internal/handler"
	"github.com/creafly/storage/internal/infra/client"
	"github.com/creafly/storage/internal/infra/database"
	"github.com/creafly/storage/internal/infra/minio"
	"github.com/jmoiron/sqlx"
	"github.com/xlab/closer"
)

type serviceProvider struct {
	config *config.Config

	db       *sqlx.DB
	migrator *database.Migrator

	minioClient *minio.Client

	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository

	fileService   *service.FileService
	folderService *service.FolderService

	fileHandler   *handler.FileHandler
	folderHandler *handler.FolderHandler
	healthHandler *handler.HealthHandler

	identityClient *client.IdentityClient
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (sp *serviceProvider) GetConfig() *config.Config {
	if sp.config == nil {
		sp.config = config.Load()
	}
	return sp.config
}

func (sp *serviceProvider) GetDB() *sqlx.DB {
	if sp.db == nil {
		cfg := sp.GetConfig()
		db, err := sqlx.Connect("postgres", cfg.Database.URL)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to connect to database")
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(1 * time.Minute)

		closer.Bind(func() {
			if err := db.Close(); err != nil {
				logger.Error().Err(err).Msg("Failed to close database connection")
			}
			logger.Info().Msg("Database connection closed")
		})

		sp.db = db
	}
	return sp.db
}

func (sp *serviceProvider) GetMigrator() *database.Migrator {
	if sp.migrator == nil {
		sp.migrator = database.NewMigrator(sp.GetDB(), "migrations")
	}
	return sp.migrator
}

func (sp *serviceProvider) GetMinioClient() *minio.Client {
	if sp.minioClient == nil {
		cfg := sp.GetConfig()
		minioClient, err := minio.NewClient(cfg.MinIO)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to create MinIO client")
		}
		sp.minioClient = minioClient
	}
	return sp.minioClient
}

func (sp *serviceProvider) EnsureMinioBucket(ctx context.Context) {
	if err := sp.GetMinioClient().EnsureBucket(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to ensure MinIO bucket")
	}
}

func (sp *serviceProvider) GetFileRepo() *repository.FileRepository {
	if sp.fileRepo == nil {
		sp.fileRepo = repository.NewFileRepository(sp.GetDB())
	}
	return sp.fileRepo
}

func (sp *serviceProvider) GetFolderRepo() *repository.FolderRepository {
	if sp.folderRepo == nil {
		sp.folderRepo = repository.NewFolderRepository(sp.GetDB())
	}
	return sp.folderRepo
}

func (sp *serviceProvider) GetFileService() *service.FileService {
	if sp.fileService == nil {
		cfg := sp.GetConfig()
		sp.fileService = service.NewFileService(
			sp.GetFileRepo(),
			sp.GetMinioClient(),
			cfg.Upload,
		)
	}
	return sp.fileService
}

func (sp *serviceProvider) GetFolderService() *service.FolderService {
	if sp.folderService == nil {
		sp.folderService = service.NewFolderService(
			sp.GetFolderRepo(),
			sp.GetFileRepo(),
		)
	}
	return sp.folderService
}

func (sp *serviceProvider) GetFileHandler() *handler.FileHandler {
	if sp.fileHandler == nil {
		sp.fileHandler = handler.NewFileHandler(sp.GetFileService())
	}
	return sp.fileHandler
}

func (sp *serviceProvider) GetFolderHandler() *handler.FolderHandler {
	if sp.folderHandler == nil {
		sp.folderHandler = handler.NewFolderHandler(sp.GetFolderService())
	}
	return sp.folderHandler
}

func (sp *serviceProvider) GetHealthHandler() *handler.HealthHandler {
	if sp.healthHandler == nil {
		sp.healthHandler = handler.NewHealthHandler(sp.GetDB())
	}
	return sp.healthHandler
}

func (sp *serviceProvider) GetIdentityClient() *client.IdentityClient {
	if sp.identityClient == nil {
		cfg := sp.GetConfig()
		sp.identityClient = client.NewIdentityClient(cfg.Identity.ServiceURL)
	}
	return sp.identityClient
}

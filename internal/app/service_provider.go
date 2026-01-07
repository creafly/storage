package app

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/creafly/logger"
	"github.com/creafly/outbox"
	"github.com/creafly/storage/internal/config"
	"github.com/creafly/storage/internal/domain/repository"
	"github.com/creafly/storage/internal/domain/service"
	"github.com/creafly/storage/internal/handler"
	"github.com/creafly/storage/internal/infra/client"
	"github.com/creafly/storage/internal/infra/database"
	"github.com/creafly/storage/internal/infra/kafka"
	"github.com/creafly/storage/internal/infra/minio"
	"github.com/jmoiron/sqlx"
	"github.com/xlab/closer"
)

type serviceProvider struct {
	config *config.Config

	db       *sqlx.DB
	migrator *database.Migrator

	kafkaProducer sarama.SyncProducer

	outboxEventHandler outbox.EventHandler
	outboxWorker       *outbox.Worker
	outboxRepo         outbox.Repository

	brandingConsumer *kafka.BrandingConsumer

	minioClient *minio.Client

	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository

	fileService   *service.FileService
	folderService *service.FolderService

	fileHandler   *handler.FileHandler
	folderHandler *handler.FolderHandler
	healthHandler *handler.HealthHandler

	identityClient *client.IdentityClient
	brandingClient *client.BrandingClient
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

func (sp *serviceProvider) GetKafkaProducer() sarama.SyncProducer {
	if sp.kafkaProducer == nil && sp.GetConfig().Kafka.Enabled && len(sp.GetConfig().Kafka.Brokers) > 0 {
		kafkaConfig := sarama.NewConfig()
		kafkaConfig.Producer.Return.Successes = true
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
		kafkaConfig.Producer.Retry.Max = 3

		producer, err := sarama.NewSyncProducer(sp.GetConfig().Kafka.Brokers, kafkaConfig)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to create Kafka producer, using noop handler")
			return nil
		}

		sp.kafkaProducer = producer

		closer.Bind(func() {
			if err := sp.kafkaProducer.Close(); err != nil {
				logger.Error().Err(err).Msg("Error closing Kafka producer")
			}
		})
	}
	return sp.kafkaProducer
}

func (sp *serviceProvider) GetOutboxEventHandler() outbox.EventHandler {
	if sp.outboxEventHandler == nil {
		if sp.GetConfig().Kafka.Enabled && sp.GetKafkaProducer() != nil {
			sp.outboxEventHandler = NewKafkaEventHandler(sp.GetKafkaProducer())
		} else {
			sp.outboxEventHandler = &outbox.NoOpHandler{}
		}
	}
	return sp.outboxEventHandler
}

func (sp *serviceProvider) GetOutboxRepo() outbox.Repository {
	if sp.outboxRepo == nil {
		sp.outboxRepo = outbox.NewPostgresRepository(sp.GetDB())
	}
	return sp.outboxRepo
}

func (sp *serviceProvider) GetOutboxWorker() *outbox.Worker {
	if sp.outboxWorker == nil {
		sp.outboxWorker = outbox.NewWorker(
			sp.GetOutboxRepo(),
			sp.GetOutboxEventHandler(),
			outbox.DefaultWorkerConfig(),
			outbox.WithLogger(logger.Log),
		)
		closer.Bind(sp.outboxWorker.Stop)
	}
	return sp.outboxWorker
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
			sp.GetOutboxRepo(),
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

func (sp *serviceProvider) GetBrandingClient() *client.BrandingClient {
	if sp.brandingClient == nil {
		cfg := sp.GetConfig()
		sp.brandingClient = client.NewBrandingClient(cfg.Branding.ServiceURL)
	}
	return sp.brandingClient
}

type KafkaEventHandler struct {
	producer sarama.SyncProducer
	topicMap map[string]string
}

func NewKafkaEventHandler(producer sarama.SyncProducer) *KafkaEventHandler {
	return &KafkaEventHandler{
		producer: producer,
		topicMap: map[string]string{
			"storage.logo_file_deleted":  "storage",
			"storage.logo_files_deleted": "storage",
		},
	}
}

func (h *KafkaEventHandler) Handle(ctx context.Context, event *outbox.Event) error {
	topic := h.getTopic(event.EventType)

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.ID.String()),
		Value: sarama.StringEncoder(event.Payload),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event_type"), Value: []byte(event.EventType)},
			{Key: []byte("event_id"), Value: []byte(event.ID.String())},
			{Key: []byte("created_at"), Value: []byte(event.CreatedAt.Format(time.RFC3339))},
		},
	}

	_, _, err := h.producer.SendMessage(msg)
	return err
}

func (h *KafkaEventHandler) getTopic(eventType string) string {
	if topic, ok := h.topicMap[eventType]; ok {
		return topic
	}
	return "events"
}

func (sp *serviceProvider) GetBrandingConsumer() *kafka.BrandingConsumer {
	if sp.brandingConsumer == nil && sp.GetConfig().Kafka.Enabled && len(sp.GetConfig().Kafka.Brokers) > 0 {
		consumer, err := kafka.NewBrandingConsumer(
			sp.GetConfig().Kafka.Brokers,
			"storage-branding-consumer",
			sp.GetFileService(),
		)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to create branding consumer")
			return nil
		}
		sp.brandingConsumer = consumer
	}
	return sp.brandingConsumer
}

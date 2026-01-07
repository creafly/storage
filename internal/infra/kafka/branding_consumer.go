package kafka

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/IBM/sarama"
	"github.com/creafly/logger"
	"github.com/creafly/storage/internal/domain/service"
	"github.com/google/uuid"
)

type BrandingConsumer struct {
	client      sarama.ConsumerGroup
	fileService *service.FileService
	topics      []string
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

func NewBrandingConsumer(brokers []string, groupID string, fileService *service.FileService) (*BrandingConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &BrandingConsumer{
		client:      client,
		fileService: fileService,
		topics:      []string{"branding"},
	}, nil
}

func (c *BrandingConsumer) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			handler := &brandingHandler{fileService: c.fileService}
			if err := c.client.Consume(ctx, c.topics, handler); err != nil {
				logger.Error().Err(err).Msg("Error consuming branding messages")
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	logger.Info().Strs("topics", c.topics).Msg("Branding consumer started")
}

func (c *BrandingConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	if err := c.client.Close(); err != nil {
		logger.Error().Err(err).Msg("Error closing branding consumer")
	}
	logger.Info().Msg("Branding consumer stopped")
}

type brandingHandler struct {
	fileService *service.FileService
}

func (h *brandingHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *brandingHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *brandingHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.processMessage(session.Context(), msg)
		session.MarkMessage(msg, "")
	}
	return nil
}

func (h *brandingHandler) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) {
	var eventType string
	for _, header := range msg.Headers {
		if string(header.Key) == "event_type" {
			eventType = string(header.Value)
			break
		}
	}

	logger.Debug().Str("topic", msg.Topic).Str("event_type", eventType).Msg("Processing branding message")

	switch eventType {
	case "branding.logo_deleted":
		h.handleLogoDeleted(ctx, msg.Value)
	case "branding.logos_deleted":
		h.handleLogosDeleted(ctx, msg.Value)
	}
}

func (h *brandingHandler) handleLogoDeleted(ctx context.Context, payload []byte) {
	var data struct {
		FileID   string `json:"file_id"`
		TenantID string `json:"tenant_id"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal logo deleted event")
		return
	}

	fileID, err := uuid.Parse(data.FileID)
	if err != nil {
		logger.Error().Err(err).Str("file_id", data.FileID).Msg("Invalid file ID")
		return
	}

	tenantID, err := uuid.Parse(data.TenantID)
	if err != nil {
		logger.Error().Err(err).Str("tenant_id", data.TenantID).Msg("Invalid tenant ID")
		return
	}

	if err := h.fileService.DeleteWithoutOutbox(ctx, fileID, tenantID); err != nil {
		logger.Error().Err(err).Str("file_id", data.FileID).Msg("Failed to delete file")
		return
	}

	logger.Info().Str("file_id", data.FileID).Msg("Successfully deleted file from branding event")
}

func (h *brandingHandler) handleLogosDeleted(ctx context.Context, payload []byte) {
	var data struct {
		FileIDs  []string `json:"file_ids"`
		TenantID string   `json:"tenant_id"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal logos deleted event")
		return
	}

	tenantID, err := uuid.Parse(data.TenantID)
	if err != nil {
		logger.Error().Err(err).Str("tenant_id", data.TenantID).Msg("Invalid tenant ID")
		return
	}

	var fileIDs []uuid.UUID
	for _, id := range data.FileIDs {
		fileID, err := uuid.Parse(id)
		if err != nil {
			logger.Error().Err(err).Str("file_id", id).Msg("Invalid file ID, skipping")
			continue
		}
		fileIDs = append(fileIDs, fileID)
	}

	if len(fileIDs) == 0 {
		return
	}

	if _, err := h.fileService.DeleteManyWithoutOutbox(ctx, fileIDs, tenantID); err != nil {
		logger.Error().Err(err).Msg("Failed to delete files")
		return
	}

	logger.Info().Int("count", len(fileIDs)).Msg("Successfully deleted files from branding event")
}

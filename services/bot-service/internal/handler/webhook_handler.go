package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/flighttracker/pkg/logger"
	"github.com/flighttracker/pkg/telegram"

	"github.com/flighttracker/services/bot-service/internal/service"
)

const telegramSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// WebhookHandler receives Telegram updates. It always acknowledges
// immediately (200 {"ok":true}) and processes the update asynchronously,
// so a slow downstream call never causes Telegram to retry-storm the
// webhook (see docs/api-contracts/bot-service.md).
type WebhookHandler struct {
	botService     *service.BotService
	webhookSecret  string
	pathSecret     string
	processTimeout time.Duration
	logger         *zap.Logger
}

func NewWebhookHandler(botService *service.BotService, webhookSecret, pathSecret string, processTimeout time.Duration, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		botService:     botService,
		webhookSecret:  webhookSecret,
		pathSecret:     pathSecret,
		processTimeout: processTimeout,
		logger:         logger,
	}
}

func (h *WebhookHandler) Handle(c *gin.Context) {
	if subtle.ConstantTimeCompare([]byte(c.Param("secret")), []byte(h.pathSecret)) != 1 {
		c.Status(http.StatusNotFound)
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.GetHeader(telegramSecretHeader)), []byte(h.webhookSecret)) != 1 {
		c.Status(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "failed to read body"})
		return
	}

	var update telegram.Update
	if err := json.Unmarshal(body, &update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid update payload"})
		return
	}

	// Acknowledge Telegram immediately; process in the background on a
	// fresh context since the request's context is cancelled the moment
	// this handler returns.
	c.JSON(http.StatusOK, gin.H{"ok": true})

	correlationID := uuid.NewString()
	go h.process(correlationID, update)
}

func (h *WebhookHandler) process(correlationID string, update telegram.Update) {
	ctx, cancel := context.WithTimeout(context.Background(), h.processTimeout)
	defer cancel()
	ctx = logger.WithCorrelationID(ctx, correlationID)

	if err := h.botService.HandleUpdate(ctx, update); err != nil {
		logger.FromContext(ctx, h.logger).Error("handle_update_failed", zap.Error(err))
	}
}

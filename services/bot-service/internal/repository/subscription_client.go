package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/flighttracker/services/bot-service/internal/api"
	"github.com/flighttracker/services/bot-service/internal/model"
)

// SubscriptionClient is bot-service's port to subscription-service's public
// CRUD API (docs/api-contracts/subscription-service.yaml). Resolving "which
// subscription id does /unsubscribe flight VN257 refer to" is deliberately
// left to the service layer (it's bot-service orchestration, not a
// subscription-service concern) — this client only exposes the primitives
// subscription-service actually publishes.
type SubscriptionClient interface {
	UpsertChat(ctx context.Context, telegramChatID int64, username, firstName, languageCode string) error
	CreateSubscription(ctx context.Context, telegramChatID int64, subType model.SubscriptionType, flightNumber, origin, destination string) (*model.Subscription, error)
	ListSubscriptions(ctx context.Context, telegramChatID int64) ([]model.Subscription, error)
	DeleteSubscription(ctx context.Context, id string, telegramChatID int64) error
}

type httpSubscriptionClient struct {
	restClient
}

func NewSubscriptionClient(baseURL string, timeout time.Duration) SubscriptionClient {
	return &httpSubscriptionClient{
		restClient: restClient{
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: timeout},
		},
	}
}

func (c *httpSubscriptionClient) UpsertChat(ctx context.Context, telegramChatID int64, username, firstName, languageCode string) error {
	req := api.UpsertChatRequestDTO{
		TelegramChatID: telegramChatID,
		Username:       username,
		FirstName:      firstName,
		LanguageCode:   languageCode,
	}
	var resp api.ChatResponseDTO
	return c.do(ctx, http.MethodPost, "/api/v1/chats", req, &resp)
}

func (c *httpSubscriptionClient) CreateSubscription(ctx context.Context, telegramChatID int64, subType model.SubscriptionType, flightNumber, origin, destination string) (*model.Subscription, error) {
	req := api.CreateSubscriptionRequestDTO{
		TelegramChatID:  telegramChatID,
		Type:            string(subType),
		FlightNumber:    flightNumber,
		OriginIATA:      origin,
		DestinationIATA: destination,
	}
	var dto api.SubscriptionResponseDTO
	if err := c.do(ctx, http.MethodPost, "/api/v1/subscriptions", req, &dto); err != nil {
		return nil, err
	}
	sub := toSubscription(dto)
	return &sub, nil
}

func (c *httpSubscriptionClient) ListSubscriptions(ctx context.Context, telegramChatID int64) ([]model.Subscription, error) {
	q := url.Values{}
	q.Set("telegram_chat_id", fmt.Sprintf("%d", telegramChatID))

	var dto api.SubscriptionListResponseDTO
	path := "/api/v1/subscriptions?" + q.Encode()
	if err := c.do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}

	subs := make([]model.Subscription, 0, len(dto.Items))
	for _, item := range dto.Items {
		subs = append(subs, toSubscription(item))
	}
	return subs, nil
}

func (c *httpSubscriptionClient) DeleteSubscription(ctx context.Context, id string, telegramChatID int64) error {
	q := url.Values{}
	q.Set("telegram_chat_id", fmt.Sprintf("%d", telegramChatID))
	path := fmt.Sprintf("/api/v1/subscriptions/%s?%s", url.PathEscape(id), q.Encode())
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func toSubscription(dto api.SubscriptionResponseDTO) model.Subscription {
	return model.Subscription{
		ID:              dto.ID,
		TelegramChatID:  dto.TelegramChatID,
		Type:            model.SubscriptionType(dto.Type),
		FlightNumber:    dto.FlightNumber,
		OriginIATA:      dto.OriginIATA,
		DestinationIATA: dto.DestinationIATA,
		Active:          dto.Active,
		CreatedAt:       dto.CreatedAt,
	}
}

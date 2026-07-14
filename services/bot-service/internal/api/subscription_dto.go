package api

import "time"

// UpsertChatRequestDTO mirrors UpsertChatRequest in
// docs/api-contracts/subscription-service.yaml.
type UpsertChatRequestDTO struct {
	TelegramChatID int64  `json:"telegram_chat_id"`
	Username       string `json:"username,omitempty"`
	FirstName      string `json:"first_name,omitempty"`
	LanguageCode   string `json:"language_code,omitempty"`
}

// ChatResponseDTO mirrors ChatResponse.
type ChatResponseDTO struct {
	ID             string    `json:"id"`
	TelegramChatID int64     `json:"telegram_chat_id"`
	Username       string    `json:"username"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateSubscriptionRequestDTO mirrors CreateSubscriptionRequest.
type CreateSubscriptionRequestDTO struct {
	TelegramChatID  int64  `json:"telegram_chat_id"`
	Type            string `json:"type"`
	FlightNumber    string `json:"flight_number,omitempty"`
	OriginIATA      string `json:"origin_iata,omitempty"`
	DestinationIATA string `json:"destination_iata,omitempty"`
}

// SubscriptionResponseDTO mirrors SubscriptionResponse.
type SubscriptionResponseDTO struct {
	ID              string    `json:"id"`
	TelegramChatID  int64     `json:"telegram_chat_id"`
	Type            string    `json:"type"`
	FlightNumber    string    `json:"flight_number"`
	OriginIATA      string    `json:"origin_iata"`
	DestinationIATA string    `json:"destination_iata"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
}

// SubscriptionListResponseDTO mirrors the GET /api/v1/subscriptions response body.
type SubscriptionListResponseDTO struct {
	Items []SubscriptionResponseDTO `json:"items"`
}

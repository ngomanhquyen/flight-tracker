package model

import "time"

// SubscriptionType mirrors subscription-service's subscription.subscription_type
// enum (docs/database/subscription-service.sql).
type SubscriptionType string

const (
	SubscriptionTypeFlight SubscriptionType = "flight"
	SubscriptionTypeRoute  SubscriptionType = "route"
)

// Subscription is bot-service's domain view of a subscription, decoded
// from subscription-service's SubscriptionResponse
// (docs/api-contracts/subscription-service.yaml).
type Subscription struct {
	ID              string
	TelegramChatID  int64
	Type            SubscriptionType
	FlightNumber    string
	OriginIATA      string
	DestinationIATA string
	Active          bool
	CreatedAt       time.Time
}

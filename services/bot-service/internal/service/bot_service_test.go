package service

import (
	"context"
	"strings"
	"testing"
	"time"

	sharederrors "github.com/flighttracker/pkg/errors"
	"github.com/flighttracker/pkg/telegram"
	"go.uber.org/zap"

	"github.com/flighttracker/services/bot-service/internal/model"
)

// fakeFlightClient and fakeSubscriptionClient let bot_service_test drive
// BotService.HandleUpdate without a network dependency, verifying the
// orchestration/dispatch logic in isolation from the HTTP transport
// implementations in internal/repository.
type fakeFlightClient struct {
	flight    *model.FlightInfo
	flightErr error
	route     []model.FlightInfo
	routeErr  error
}

func (f *fakeFlightClient) GetByFlightNumber(_ context.Context, _ string) (*model.FlightInfo, error) {
	return f.flight, f.flightErr
}

func (f *fakeFlightClient) SearchByRoute(_ context.Context, _, _ string, _ int) ([]model.FlightInfo, error) {
	return f.route, f.routeErr
}

type fakeSubscriptionClient struct {
	subs      []model.Subscription
	createErr error
	deleteErr error
	deletedID string
}

func (f *fakeSubscriptionClient) UpsertChat(_ context.Context, _ int64, _, _, _ string) error {
	return nil
}

func (f *fakeSubscriptionClient) CreateSubscription(_ context.Context, chatID int64, subType model.SubscriptionType, flightNumber, origin, destination string) (*model.Subscription, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &model.Subscription{ID: "new-id", TelegramChatID: chatID, Type: subType, FlightNumber: flightNumber, OriginIATA: origin, DestinationIATA: destination}, nil
}

func (f *fakeSubscriptionClient) ListSubscriptions(_ context.Context, _ int64) ([]model.Subscription, error) {
	return f.subs, nil
}

func (f *fakeSubscriptionClient) DeleteSubscription(_ context.Context, id string, _ int64) error {
	f.deletedID = id
	return f.deleteErr
}

type fakeSender struct {
	chatID int64
	text   string
}

func (f *fakeSender) SendMessage(_ context.Context, chatID int64, text string) error {
	f.chatID = chatID
	f.text = text
	return nil
}

func newUpdate(text string) telegram.Update {
	return telegram.Update{
		Message: &telegram.Message{
			Chat: telegram.Chat{ID: 42},
			From: &telegram.User{ID: 1, Username: "alice"},
			Text: text,
		},
	}
}

func TestBotService_FlightLookup_Found(t *testing.T) {
	dep := time.Date(2026, 7, 9, 7, 30, 0, 0, time.UTC)
	fc := &fakeFlightClient{flight: &model.FlightInfo{
		FlightNumber: "VN257", OriginIATA: "HAN", DestinationIATA: "SGN",
		Status: model.StatusBoarding, ScheduledDeparture: dep,
	}}
	sc := &fakeSubscriptionClient{}
	sender := &fakeSender{}

	svc := NewBotService(fc, sc, sender, zap.NewNop())
	if err := svc.HandleUpdate(context.Background(), newUpdate("/flight VN257")); err != nil {
		t.Fatalf("HandleUpdate error: %v", err)
	}

	if sender.chatID != 42 {
		t.Errorf("chatID = %d, want 42", sender.chatID)
	}
	if !strings.Contains(sender.text, "VN257") || !strings.Contains(sender.text, "BOARDING") {
		t.Errorf("reply = %q, want it to mention VN257 and BOARDING", sender.text)
	}
}

func TestBotService_FlightLookup_NotFound(t *testing.T) {
	fc := &fakeFlightClient{flightErr: sharederrors.NotFound("FLIGHT_NOT_FOUND", "no such flight")}
	sc := &fakeSubscriptionClient{}
	sender := &fakeSender{}

	svc := NewBotService(fc, sc, sender, zap.NewNop())
	_ = svc.HandleUpdate(context.Background(), newUpdate("/flight VN999"))

	if !strings.Contains(sender.text, "No flight found") {
		t.Errorf("reply = %q, want a not-found message", sender.text)
	}
}

func TestBotService_SubscribeFlight_Duplicate(t *testing.T) {
	fc := &fakeFlightClient{}
	sc := &fakeSubscriptionClient{createErr: sharederrors.Conflict("DUPLICATE_SUBSCRIPTION", "already exists")}
	sender := &fakeSender{}

	svc := NewBotService(fc, sc, sender, zap.NewNop())
	_ = svc.HandleUpdate(context.Background(), newUpdate("/subscribe flight VN257"))

	if !strings.Contains(sender.text, "already subscribed") {
		t.Errorf("reply = %q, want an already-subscribed message", sender.text)
	}
}

func TestBotService_UnsubscribeFlight_ResolvesIDFromList(t *testing.T) {
	fc := &fakeFlightClient{}
	sc := &fakeSubscriptionClient{subs: []model.Subscription{
		{ID: "sub-1", Type: model.SubscriptionTypeFlight, FlightNumber: "VN257"},
		{ID: "sub-2", Type: model.SubscriptionTypeRoute, OriginIATA: "HAN", DestinationIATA: "SGN"},
	}}
	sender := &fakeSender{}

	svc := NewBotService(fc, sc, sender, zap.NewNop())
	_ = svc.HandleUpdate(context.Background(), newUpdate("/unsubscribe flight VN257"))

	if sc.deletedID != "sub-1" {
		t.Errorf("deletedID = %q, want sub-1", sc.deletedID)
	}
	if !strings.Contains(sender.text, "Unsubscribed") {
		t.Errorf("reply = %q, want an unsubscribed confirmation", sender.text)
	}
}

func TestBotService_UnsubscribeFlight_NotSubscribed(t *testing.T) {
	fc := &fakeFlightClient{}
	sc := &fakeSubscriptionClient{}
	sender := &fakeSender{}

	svc := NewBotService(fc, sc, sender, zap.NewNop())
	_ = svc.HandleUpdate(context.Background(), newUpdate("/unsubscribe flight VN257"))

	if sc.deletedID != "" {
		t.Errorf("expected no delete call, got deletedID = %q", sc.deletedID)
	}
	if !strings.Contains(sender.text, "not subscribed") {
		t.Errorf("reply = %q, want a not-subscribed message", sender.text)
	}
}

func TestBotService_UsageError_SkipsDownstreamCall(t *testing.T) {
	fc := &fakeFlightClient{flightErr: sharederrors.Internal("SHOULD_NOT_BE_CALLED", "", nil)}
	sc := &fakeSubscriptionClient{}
	sender := &fakeSender{}

	svc := NewBotService(fc, sc, sender, zap.NewNop())
	_ = svc.HandleUpdate(context.Background(), newUpdate("/flight not-a-flight-number"))

	if !strings.Contains(sender.text, "Usage:") {
		t.Errorf("reply = %q, want a usage hint", sender.text)
	}
}

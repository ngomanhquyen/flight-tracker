// Package service holds bot-service's use cases: parsing Telegram commands
// and orchestrating calls to flight-service/subscription-service to
// produce a reply.
package service

import (
	"context"
	"errors"
	"net/http"

	sharederrors "github.com/flighttracker/pkg/errors"
	"github.com/flighttracker/pkg/logger"
	"github.com/flighttracker/pkg/telegram"
	"go.uber.org/zap"

	"github.com/flighttracker/services/bot-service/internal/model"
	"github.com/flighttracker/services/bot-service/internal/repository"
)

// BotService dispatches parsed Telegram commands to flight-service /
// subscription-service and sends the resulting reply back to the chat.
type BotService struct {
	flightClient       repository.FlightClient
	subscriptionClient repository.SubscriptionClient
	parser             *CommandParser
	sender             telegram.Sender
	logger             *zap.Logger
}

func NewBotService(
	flightClient repository.FlightClient,
	subscriptionClient repository.SubscriptionClient,
	sender telegram.Sender,
	logger *zap.Logger,
) *BotService {
	return &BotService{
		flightClient:       flightClient,
		subscriptionClient: subscriptionClient,
		parser:             NewCommandParser(),
		sender:             sender,
		logger:             logger,
	}
}

// HandleUpdate processes one Telegram update end-to-end: parses the
// command, calls downstream services, and sends the reply. Called from a
// goroutine spawned by the webhook handler, so it owns its own context
// (not the original HTTP request's, which is already closed by the time
// this runs) and never returns a value Telegram is waiting on.
func (s *BotService) HandleUpdate(ctx context.Context, update telegram.Update) error {
	if update.Message == nil || update.Message.Text == "" {
		return nil
	}
	msg := update.Message
	chatID := msg.Chat.ID
	log := logger.FromContext(ctx, s.logger).With(zap.Int64("chat_id", chatID))

	// Upsert the chat on every message (not just /start): cheap and
	// idempotent, and guarantees subscription-service always has a chat
	// row to attach a subscription to, even if a user runs /subscribe
	// without ever having sent /start.
	if msg.From != nil {
		if err := s.subscriptionClient.UpsertChat(ctx, chatID, msg.From.Username, msg.From.FirstName, msg.From.LanguageCode); err != nil {
			log.Warn("upsert_chat_failed", zap.Error(err))
		}
	}

	parsed := s.parser.Parse(msg.Text)
	reply := s.dispatch(ctx, log, chatID, parsed)

	if err := s.sender.SendMessage(ctx, chatID, reply); err != nil {
		log.Error("send_message_failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *BotService) dispatch(ctx context.Context, log *zap.Logger, chatID int64, cmd model.ParsedCommand) string {
	if cmd.UsageError != "" {
		return cmd.UsageError
	}

	switch cmd.Type {
	case model.CommandStart:
		return welcomeText
	case model.CommandHelp:
		return helpText
	case model.CommandFlight:
		return s.handleFlightLookup(ctx, log, cmd.FlightNumber)
	case model.CommandRoute:
		return s.handleRouteSearch(ctx, log, cmd.Origin, cmd.Destination)
	case model.CommandSubscribeFlight:
		return s.handleSubscribeFlight(ctx, log, chatID, cmd.FlightNumber)
	case model.CommandSubscribeRoute:
		return s.handleSubscribeRoute(ctx, log, chatID, cmd.Origin, cmd.Destination)
	case model.CommandUnsubscribeFlight:
		return s.handleUnsubscribeFlight(ctx, log, chatID, cmd.FlightNumber)
	case model.CommandUnsubscribeRoute:
		return s.handleUnsubscribeRoute(ctx, log, chatID, cmd.Origin, cmd.Destination)
	case model.CommandListSubscriptions:
		return s.handleListSubscriptions(ctx, log, chatID)
	default:
		return "Unknown command. Send /help to see what I can do."
	}
}

func (s *BotService) handleFlightLookup(ctx context.Context, log *zap.Logger, flightNumber string) string {
	info, err := s.flightClient.GetByFlightNumber(ctx, flightNumber)
	if err != nil {
		if isNotFound(err) {
			return "No flight found for " + flightNumber + " today."
		}
		log.Error("flight_lookup_failed", zap.String("flight_number", flightNumber), zap.Error(err))
		return "Sorry, I couldn't look that up right now. Please try again shortly."
	}
	return formatFlight(*info)
}

func (s *BotService) handleRouteSearch(ctx context.Context, log *zap.Logger, origin, destination string) string {
	flights, err := s.flightClient.SearchByRoute(ctx, origin, destination, 10)
	if err != nil {
		log.Error("route_search_failed", zap.String("origin", origin), zap.String("destination", destination), zap.Error(err))
		return "Sorry, I couldn't search that route right now. Please try again shortly."
	}
	return formatRouteResults(origin, destination, flights)
}

func (s *BotService) handleSubscribeFlight(ctx context.Context, log *zap.Logger, chatID int64, flightNumber string) string {
	_, err := s.subscriptionClient.CreateSubscription(ctx, chatID, model.SubscriptionTypeFlight, flightNumber, "", "")
	if err != nil {
		if isConflict(err) {
			return "You're already subscribed to flight " + flightNumber + "."
		}
		log.Error("subscribe_flight_failed", zap.String("flight_number", flightNumber), zap.Error(err))
		return "Sorry, I couldn't create that subscription right now. Please try again shortly."
	}
	return "✅ Subscribed to flight " + flightNumber + ". You'll get a message when its status changes."
}

func (s *BotService) handleSubscribeRoute(ctx context.Context, log *zap.Logger, chatID int64, origin, destination string) string {
	_, err := s.subscriptionClient.CreateSubscription(ctx, chatID, model.SubscriptionTypeRoute, "", origin, destination)
	if err != nil {
		if isConflict(err) {
			return "You're already subscribed to route " + origin + " → " + destination + "."
		}
		log.Error("subscribe_route_failed", zap.String("origin", origin), zap.String("destination", destination), zap.Error(err))
		return "Sorry, I couldn't create that subscription right now. Please try again shortly."
	}
	return "✅ Subscribed to route " + origin + " → " + destination + ". You'll get a message when any matching flight's status changes."
}

func (s *BotService) handleUnsubscribeFlight(ctx context.Context, log *zap.Logger, chatID int64, flightNumber string) string {
	id, err := s.findSubscriptionID(ctx, chatID, func(sub model.Subscription) bool {
		return sub.Type == model.SubscriptionTypeFlight && sub.FlightNumber == flightNumber
	})
	if err != nil {
		log.Error("unsubscribe_flight_lookup_failed", zap.String("flight_number", flightNumber), zap.Error(err))
		return "Sorry, I couldn't process that right now. Please try again shortly."
	}
	if id == "" {
		return "You're not subscribed to flight " + flightNumber + "."
	}
	if err := s.subscriptionClient.DeleteSubscription(ctx, id, chatID); err != nil {
		log.Error("unsubscribe_flight_failed", zap.String("flight_number", flightNumber), zap.Error(err))
		return "Sorry, I couldn't remove that subscription right now. Please try again shortly."
	}
	return "🗑 Unsubscribed from flight " + flightNumber + "."
}

func (s *BotService) handleUnsubscribeRoute(ctx context.Context, log *zap.Logger, chatID int64, origin, destination string) string {
	id, err := s.findSubscriptionID(ctx, chatID, func(sub model.Subscription) bool {
		return sub.Type == model.SubscriptionTypeRoute && sub.OriginIATA == origin && sub.DestinationIATA == destination
	})
	if err != nil {
		log.Error("unsubscribe_route_lookup_failed", zap.String("origin", origin), zap.String("destination", destination), zap.Error(err))
		return "Sorry, I couldn't process that right now. Please try again shortly."
	}
	if id == "" {
		return "You're not subscribed to route " + origin + " → " + destination + "."
	}
	if err := s.subscriptionClient.DeleteSubscription(ctx, id, chatID); err != nil {
		log.Error("unsubscribe_route_failed", zap.String("origin", origin), zap.String("destination", destination), zap.Error(err))
		return "Sorry, I couldn't remove that subscription right now. Please try again shortly."
	}
	return "🗑 Unsubscribed from route " + origin + " → " + destination + "."
}

func (s *BotService) handleListSubscriptions(ctx context.Context, log *zap.Logger, chatID int64) string {
	subs, err := s.subscriptionClient.ListSubscriptions(ctx, chatID)
	if err != nil {
		log.Error("list_subscriptions_failed", zap.Error(err))
		return "Sorry, I couldn't fetch your subscriptions right now. Please try again shortly."
	}
	return formatSubscriptions(subs)
}

// findSubscriptionID resolves a subscription id from a flight/route match
// by listing the chat's active subscriptions — bot-service orchestration,
// since subscription-service's DELETE endpoint takes an id, not a
// flight/route (see docs/api-contracts/subscription-service.yaml).
func (s *BotService) findSubscriptionID(ctx context.Context, chatID int64, match func(model.Subscription) bool) (string, error) {
	subs, err := s.subscriptionClient.ListSubscriptions(ctx, chatID)
	if err != nil {
		return "", err
	}
	for _, sub := range subs {
		if match(sub) {
			return sub.ID, nil
		}
	}
	return "", nil
}

func isNotFound(err error) bool {
	var appErr *sharederrors.AppError
	return errors.As(err, &appErr) && appErr.HTTPStatus == http.StatusNotFound
}

func isConflict(err error) bool {
	var appErr *sharederrors.AppError
	return errors.As(err, &appErr) && appErr.HTTPStatus == http.StatusConflict
}

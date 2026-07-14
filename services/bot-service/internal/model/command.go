package model

// CommandType identifies which Telegram command a message maps to, per the
// grammar in docs/api-contracts/bot-service.md.
type CommandType string

const (
	CommandStart             CommandType = "start"
	CommandHelp              CommandType = "help"
	CommandFlight            CommandType = "flight"
	CommandRoute             CommandType = "route"
	CommandSubscribeFlight   CommandType = "subscribe_flight"
	CommandSubscribeRoute    CommandType = "subscribe_route"
	CommandUnsubscribeFlight CommandType = "unsubscribe_flight"
	CommandUnsubscribeRoute  CommandType = "unsubscribe_route"
	CommandListSubscriptions CommandType = "subscriptions"
	CommandUnknown           CommandType = "unknown"
)

// ParsedCommand is the normalized result of parsing a raw Telegram message
// into a command the service layer can dispatch on.
type ParsedCommand struct {
	Type         CommandType
	FlightNumber string
	Origin       string
	Destination  string
	UsageError   string // set when args were malformed; dispatch should reply with this instead of calling downstream services
}

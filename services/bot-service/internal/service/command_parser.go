package service

import (
	"strings"

	sharedvalidator "github.com/flighttracker/pkg/validator"

	"github.com/flighttracker/services/bot-service/internal/model"
)

// CommandParser turns raw Telegram message text into a model.ParsedCommand,
// per the grammar documented in docs/api-contracts/bot-service.md.
type CommandParser struct{}

func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

func (p *CommandParser) Parse(text string) model.ParsedCommand {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return model.ParsedCommand{Type: model.CommandUnknown}
	}

	keyword := normalizeKeyword(fields[0])
	args := fields[1:]

	switch keyword {
	case "/start":
		return model.ParsedCommand{Type: model.CommandStart}
	case "/help":
		return model.ParsedCommand{Type: model.CommandHelp}
	case "/flight":
		return parseFlight(args)
	case "/route":
		return parseRoute(args, model.CommandRoute)
	case "/subscribe":
		return parseSubscribe(args)
	case "/unsubscribe":
		return parseUnsubscribe(args)
	case "/subscriptions":
		return model.ParsedCommand{Type: model.CommandListSubscriptions}
	default:
		return model.ParsedCommand{Type: model.CommandUnknown}
	}
}

// normalizeKeyword lower-cases the command keyword and strips any
// "@botusername" suffix Telegram appends in group chats (e.g.
// "/flight@FlightTrackerBot" -> "/flight").
func normalizeKeyword(keyword string) string {
	keyword = strings.ToLower(keyword)
	if idx := strings.Index(keyword, "@"); idx != -1 {
		keyword = keyword[:idx]
	}
	return keyword
}

func parseFlight(args []string) model.ParsedCommand {
	if len(args) != 1 || !sharedvalidator.IsFlightNumber(args[0]) {
		return model.ParsedCommand{
			Type:       model.CommandFlight,
			UsageError: "Usage: /flight <FLIGHT_NUMBER>, e.g. /flight VN257",
		}
	}
	return model.ParsedCommand{Type: model.CommandFlight, FlightNumber: strings.ToUpper(args[0])}
}

func parseRoute(args []string, cmdType model.CommandType) model.ParsedCommand {
	usage := "Usage: /route <ORIGIN> <DESTINATION>, e.g. /route HAN SGN"
	if cmdType != model.CommandRoute {
		usage = "Usage: /subscribe route <ORIGIN> <DESTINATION>, e.g. /subscribe route HAN SGN"
	}
	if len(args) != 2 || !sharedvalidator.IsIATA(args[0]) || !sharedvalidator.IsIATA(args[1]) {
		return model.ParsedCommand{Type: cmdType, UsageError: usage}
	}
	return model.ParsedCommand{
		Type:        cmdType,
		Origin:      strings.ToUpper(args[0]),
		Destination: strings.ToUpper(args[1]),
	}
}

func parseSubscribe(args []string) model.ParsedCommand {
	usage := "Usage: /subscribe flight <FLIGHT_NUMBER> or /subscribe route <ORIGIN> <DESTINATION>"
	if len(args) < 2 {
		return model.ParsedCommand{Type: model.CommandSubscribeFlight, UsageError: usage}
	}
	switch strings.ToLower(args[0]) {
	case "flight":
		if len(args) != 2 || !sharedvalidator.IsFlightNumber(args[1]) {
			return model.ParsedCommand{
				Type:       model.CommandSubscribeFlight,
				UsageError: "Usage: /subscribe flight <FLIGHT_NUMBER>, e.g. /subscribe flight VN257",
			}
		}
		return model.ParsedCommand{Type: model.CommandSubscribeFlight, FlightNumber: strings.ToUpper(args[1])}
	case "route":
		return parseRoute(args[1:], model.CommandSubscribeRoute)
	default:
		return model.ParsedCommand{Type: model.CommandSubscribeFlight, UsageError: usage}
	}
}

func parseUnsubscribe(args []string) model.ParsedCommand {
	usage := "Usage: /unsubscribe flight <FLIGHT_NUMBER> or /unsubscribe route <ORIGIN> <DESTINATION>"
	if len(args) < 2 {
		return model.ParsedCommand{Type: model.CommandUnsubscribeFlight, UsageError: usage}
	}
	switch strings.ToLower(args[0]) {
	case "flight":
		if len(args) != 2 || !sharedvalidator.IsFlightNumber(args[1]) {
			return model.ParsedCommand{
				Type:       model.CommandUnsubscribeFlight,
				UsageError: "Usage: /unsubscribe flight <FLIGHT_NUMBER>, e.g. /unsubscribe flight VN257",
			}
		}
		return model.ParsedCommand{Type: model.CommandUnsubscribeFlight, FlightNumber: strings.ToUpper(args[1])}
	case "route":
		if len(args) != 3 || !sharedvalidator.IsIATA(args[1]) || !sharedvalidator.IsIATA(args[2]) {
			return model.ParsedCommand{
				Type:       model.CommandUnsubscribeRoute,
				UsageError: "Usage: /unsubscribe route <ORIGIN> <DESTINATION>, e.g. /unsubscribe route HAN SGN",
			}
		}
		return model.ParsedCommand{
			Type:        model.CommandUnsubscribeRoute,
			Origin:      strings.ToUpper(args[1]),
			Destination: strings.ToUpper(args[2]),
		}
	default:
		return model.ParsedCommand{Type: model.CommandUnsubscribeFlight, UsageError: usage}
	}
}

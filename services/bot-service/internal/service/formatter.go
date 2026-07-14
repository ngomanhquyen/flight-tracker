package service

import (
	"fmt"
	"strings"

	"github.com/flighttracker/services/bot-service/internal/model"
)

const helpText = `Available commands:
/flight <FLIGHT_NUMBER> - look up a flight, e.g. /flight VN257
/route <ORIGIN> <DEST> - search flights on a route, e.g. /route HAN SGN
/subscribe flight <FLIGHT_NUMBER> - get notified about a flight
/subscribe route <ORIGIN> <DEST> - get notified about a route
/unsubscribe flight <FLIGHT_NUMBER>
/unsubscribe route <ORIGIN> <DEST>
/subscriptions - list your subscriptions
/help - show this message`

const welcomeText = "👋 Welcome to Flight Tracker Bot!\n\n" + helpText

func formatFlight(f model.FlightInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "✈ %s", f.FlightNumber)
	if f.AirlineName != "" {
		fmt.Fprintf(&b, " (%s)", f.AirlineName)
	}
	fmt.Fprintf(&b, "\n%s → %s\n", f.OriginIATA, f.DestinationIATA)
	fmt.Fprintf(&b, "Status: %s\n", f.Status)
	if f.Gate != "" || f.Terminal != "" {
		fmt.Fprintf(&b, "Gate: %s  Terminal: %s\n", orDash(f.Gate), orDash(f.Terminal))
	}
	fmt.Fprintf(&b, "Scheduled dep: %s", formatClock(f.ScheduledDeparture.Hour(), f.ScheduledDeparture.Minute()))
	if f.EstimatedDeparture != nil {
		fmt.Fprintf(&b, "  Est: %s", formatClock(f.EstimatedDeparture.Hour(), f.EstimatedDeparture.Minute()))
	}
	return b.String()
}

func formatRouteResults(origin, destination string, flights []model.FlightInfo) string {
	if len(flights) == 0 {
		return fmt.Sprintf("No flights found for %s → %s today.", origin, destination)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Flights %s → %s:\n\n", origin, destination)
	for i, f := range flights {
		fmt.Fprintf(&b, "%d. %s  %s  dep %s\n", i+1, f.FlightNumber, f.Status, formatClock(f.ScheduledDeparture.Hour(), f.ScheduledDeparture.Minute()))
	}
	return strings.TrimSpace(b.String())
}

func formatSubscriptions(subs []model.Subscription) string {
	if len(subs) == 0 {
		return "You have no active subscriptions. Use /subscribe flight <NUM> or /subscribe route <ORIG> <DEST>."
	}
	var b strings.Builder
	b.WriteString("Your subscriptions:\n")
	for i, s := range subs {
		switch s.Type {
		case model.SubscriptionTypeFlight:
			fmt.Fprintf(&b, "%d. Flight %s\n", i+1, s.FlightNumber)
		case model.SubscriptionTypeRoute:
			fmt.Fprintf(&b, "%d. Route %s → %s\n", i+1, s.OriginIATA, s.DestinationIATA)
		}
	}
	b.WriteString("Use /unsubscribe flight <NUM> or /unsubscribe route <ORIG> <DEST> to remove.")
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func formatClock(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

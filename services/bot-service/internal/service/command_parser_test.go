package service

import (
	"testing"

	"github.com/flighttracker/services/bot-service/internal/model"
)

func TestCommandParser_Parse(t *testing.T) {
	p := NewCommandParser()

	tests := []struct {
		name       string
		input      string
		wantType   model.CommandType
		wantFlight string
		wantOrigin string
		wantDest   string
		wantUsage  bool
	}{
		{name: "start", input: "/start", wantType: model.CommandStart},
		{name: "help", input: "/help", wantType: model.CommandHelp},
		{name: "flight lowercase", input: "/flight vn257", wantType: model.CommandFlight, wantFlight: "VN257"},
		{name: "flight with bot suffix", input: "/flight@FlightTrackerBot VN257", wantType: model.CommandFlight, wantFlight: "VN257"},
		{name: "flight missing arg", input: "/flight", wantType: model.CommandFlight, wantUsage: true},
		{name: "flight invalid format", input: "/flight 257VN", wantType: model.CommandFlight, wantUsage: true},
		{name: "route", input: "/route han sgn", wantType: model.CommandRoute, wantOrigin: "HAN", wantDest: "SGN"},
		{name: "route missing arg", input: "/route HAN", wantType: model.CommandRoute, wantUsage: true},
		{name: "route invalid iata", input: "/route HANOI SGN", wantType: model.CommandRoute, wantUsage: true},
		{name: "subscribe flight", input: "/subscribe flight VN257", wantType: model.CommandSubscribeFlight, wantFlight: "VN257"},
		{name: "subscribe route", input: "/subscribe route HAN SGN", wantType: model.CommandSubscribeRoute, wantOrigin: "HAN", wantDest: "SGN"},
		{name: "subscribe unknown kind", input: "/subscribe airport HAN", wantType: model.CommandSubscribeFlight, wantUsage: true},
		{name: "unsubscribe flight", input: "/unsubscribe flight VN257", wantType: model.CommandUnsubscribeFlight, wantFlight: "VN257"},
		{name: "unsubscribe route", input: "/unsubscribe route HAN SGN", wantType: model.CommandUnsubscribeRoute, wantOrigin: "HAN", wantDest: "SGN"},
		{name: "subscriptions", input: "/subscriptions", wantType: model.CommandListSubscriptions},
		{name: "unknown command", input: "/foo bar", wantType: model.CommandUnknown},
		{name: "empty", input: "", wantType: model.CommandUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Parse(tt.input)
			if got.Type != tt.wantType {
				t.Fatalf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if (got.UsageError != "") != tt.wantUsage {
				t.Fatalf("UsageError set = %v, want %v (usage: %q)", got.UsageError != "", tt.wantUsage, got.UsageError)
			}
			if tt.wantUsage {
				return
			}
			if got.FlightNumber != tt.wantFlight {
				t.Errorf("FlightNumber = %q, want %q", got.FlightNumber, tt.wantFlight)
			}
			if got.Origin != tt.wantOrigin {
				t.Errorf("Origin = %q, want %q", got.Origin, tt.wantOrigin)
			}
			if got.Destination != tt.wantDest {
				t.Errorf("Destination = %q, want %q", got.Destination, tt.wantDest)
			}
		})
	}
}

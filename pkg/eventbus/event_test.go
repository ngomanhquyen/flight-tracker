package eventbus

import "testing"

func TestEventType_RoutingKey(t *testing.T) {
	tests := map[EventType]string{
		EventFlightCreated:   "flight.created",
		EventFlightUpdated:   "flight.updated",
		EventFlightDelayed:   "flight.delayed",
		EventFlightBoarding:  "flight.boarding",
		EventFlightDeparted:  "flight.departed",
		EventFlightLanded:    "flight.landed",
		EventFlightCancelled: "flight.cancelled",
	}

	for eventType, want := range tests {
		if got := eventType.RoutingKey(); got != want {
			t.Errorf("%s.RoutingKey() = %q, want %q", eventType, got, want)
		}
	}
}

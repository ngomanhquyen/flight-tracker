package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// PublisherConfig configures a Publisher.
type PublisherConfig struct {
	URL      string // amqp:// connection string
	Exchange string // topic exchange name, e.g. "flight.events"
}

// Publisher publishes FlightEvents to a durable topic exchange with
// publisher confirms, matching docs/events/event-catalog.md's delivery
// guarantee ("failed publishes are logged and retried by the next poll
// cycle at worst").
type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	confirms chan amqp.Confirmation
	returns  chan amqp.Return
}

// NewPublisher dials url, opens a confirm-mode channel, and idempotently
// declares the durable topic exchange.
func NewPublisher(cfg PublisherConfig, log *zap.Logger) (*Publisher, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("eventbus: dial: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("eventbus: open channel: %w", err)
	}

	if err := channel.Confirm(false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("eventbus: enable publisher confirms: %w", err)
	}

	if err := channel.ExchangeDeclare(cfg.Exchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("eventbus: declare exchange %s: %w", cfg.Exchange, err)
	}

	p := &Publisher{
		conn:     conn,
		channel:  channel,
		exchange: cfg.Exchange,
		confirms: channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
		returns:  channel.NotifyReturn(make(chan amqp.Return, 1)),
	}

	// Publishing with mandatory=true (see Publish) means an unroutable
	// message is sent back on this channel instead of just being dropped;
	// drain it so a returned message can never block a future publish.
	go func() {
		for ret := range p.returns {
			log.Warn("eventbus_message_returned",
				zap.String("exchange", ret.Exchange),
				zap.String("routing_key", ret.RoutingKey),
				zap.String("reply_text", ret.ReplyText),
			)
		}
	}()

	return p, nil
}

// Publish marshals event to JSON and publishes it under
// event.EventType.RoutingKey(), blocking until the broker confirms receipt
// or ctx is done.
func (p *Publisher) Publish(ctx context.Context, event FlightEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("eventbus: marshal event %s: %w", event.EventID, err)
	}

	err = p.channel.PublishWithContext(ctx, p.exchange, event.EventType.RoutingKey(), true, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    event.EventID,
		Timestamp:    time.Now(),
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("eventbus: publish %s: %w", event.EventID, err)
	}

	select {
	case confirm := <-p.confirms:
		if !confirm.Ack {
			return fmt.Errorf("eventbus: broker nacked publish of event %s", event.EventID)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the channel and connection.
func (p *Publisher) Close() error {
	chErr := p.channel.Close()
	connErr := p.conn.Close()
	if chErr != nil {
		return chErr
	}
	return connErr
}

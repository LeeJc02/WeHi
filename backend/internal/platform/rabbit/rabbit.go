package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/platform/observability"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/extra/redisotel/v9"
	redispkg "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

type Client struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	redis       *redispkg.Client
	exchange    string
	useRedisBus bool
}

// New keeps one messaging interface for both broker-backed topics and the
// lightweight Redis Pub/Sub mode used by local environments.
func New(url, exchange string) (*Client, error) {
	if strings.HasPrefix(url, "redis://") {
		parsed, err := urlpkg(url)
		if err != nil {
			return nil, err
		}
		client := redispkg.NewClient(&redispkg.Options{
			Addr: parsed.Host,
		})
		_ = redisotel.InstrumentTracing(client)
		if err := client.Ping(context.Background()).Err(); err != nil {
			return nil, err
		}
		return &Client{
			redis:       client,
			exchange:    exchange,
			useRedisBus: true,
		}, nil
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &Client{conn: conn, channel: ch, exchange: exchange}, nil
}

func (c *Client) Close() error {
	if c.useRedisBus {
		if c.redis != nil {
			return c.redis.Close()
		}
		return nil
	}
	var err error
	if c.channel != nil {
		err = c.channel.Close()
	}
	if c.conn != nil {
		if closeErr := c.conn.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (c *Client) PublishJSON(routingKey string, payload any) error {
	return c.PublishJSONWithContext(context.Background(), routingKey, payload)
}

// PublishJSONWithContext preserves trace context on AMQP publishes and uses the
// same routing-key contract when the client falls back to Redis channels.
func (c *Client) PublishJSONWithContext(ctx context.Context, routingKey string, payload any) error {
	if c == nil {
		return nil
	}
	ctx, span := observability.Tracer("rabbitmq").Start(ctx, "rabbit.publish")
	defer span.End()
	span.SetAttributes(attribute.String("messaging.destination", c.exchange), attribute.String("messaging.routing_key", routingKey))
	if c.useRedisBus {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return c.redis.Publish(ctx, c.redisChannel(routingKey), body).Err()
	}
	if c.channel == nil {
		return errors.New("rabbit client not initialized")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	headers := amqp.Table{}
	otel.GetTextMapPropagator().Inject(ctx, amqpHeaderCarrier(headers))
	return c.channel.PublishWithContext(pubCtx, c.exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
		Headers:     headers,
	})
}

// Consume binds one handler to every requested routing key so downstream
// services can fan out lifecycle events without knowing the transport details.
func (c *Client) Consume(queue string, bindings []string, handler func(ctx context.Context, routingKey string, body []byte) error) error {
	if c.useRedisBus {
		channels := make([]string, 0, len(bindings))
		for _, binding := range bindings {
			channels = append(channels, c.redisChannel(binding))
		}
		pubsub := c.redis.Subscribe(context.Background(), channels...)
		if _, err := pubsub.Receive(context.Background()); err != nil {
			return err
		}
		go func() {
			for message := range pubsub.Channel() {
				key := strings.TrimPrefix(message.Channel, c.exchange+".")
				ctx, span := observability.Tracer("rabbitmq").Start(context.Background(), "redis_pubsub.consume")
				span.SetAttributes(attribute.String("messaging.destination", c.exchange), attribute.String("messaging.routing_key", key))
				_ = handler(ctx, key, []byte(message.Payload))
				span.End()
			}
		}()
		return nil
	}
	if _, err := c.channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	for _, key := range bindings {
		if err := c.channel.QueueBind(queue, key, c.exchange, false, nil); err != nil {
			return err
		}
	}
	msgs, err := c.channel.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for msg := range msgs {
			ctx := otel.GetTextMapPropagator().Extract(context.Background(), amqpHeaderCarrier(msg.Headers))
			ctx, span := observability.Tracer("rabbitmq").Start(ctx, "rabbit.consume")
			span.SetAttributes(attribute.String("messaging.destination", c.exchange), attribute.String("messaging.routing_key", msg.RoutingKey))
			if err := handler(ctx, msg.RoutingKey, msg.Body); err != nil {
				span.End()
				_ = msg.Nack(false, true)
				continue
			}
			span.End()
			_ = msg.Ack(false)
		}
	}()
	return nil
}

func (c *Client) redisChannel(routingKey string) string {
	return c.exchange + "." + routingKey
}

func urlpkg(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

type amqpHeaderCarrier amqp.Table

func (c amqpHeaderCarrier) Get(key string) string {
	value, ok := c[key]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func (c amqpHeaderCarrier) Set(key, value string) {
	c[key] = value
}

func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = amqpHeaderCarrier{}

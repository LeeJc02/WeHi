package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	redispkg "github.com/redis/go-redis/v9"
)

type Client struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	redis       *redispkg.Client
	exchange    string
	useRedisBus bool
}

func New(url, exchange string) (*Client, error) {
	if strings.HasPrefix(url, "redis://") {
		parsed, err := urlpkg(url)
		if err != nil {
			return nil, err
		}
		client := redispkg.NewClient(&redispkg.Options{
			Addr: parsed.Host,
		})
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
	if c == nil {
		return nil
	}
	if c.useRedisBus {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return c.redis.Publish(context.Background(), c.redisChannel(routingKey), body).Err()
	}
	if c.channel == nil {
		return errors.New("rabbit client not initialized")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.channel.PublishWithContext(ctx, c.exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
	})
}

func (c *Client) Consume(queue string, bindings []string, handler func(routingKey string, body []byte) error) error {
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
				_ = handler(key, []byte(message.Payload))
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
			if err := handler(msg.RoutingKey, msg.Body); err != nil {
				_ = msg.Nack(false, true)
				continue
			}
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

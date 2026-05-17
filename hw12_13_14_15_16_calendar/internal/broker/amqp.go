package broker

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPBroker реализует интерфейс Broker с использованием официального AMQP клиента.
type AMQPBroker struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewAMQPBroker создаёт экземпляр брокера без подключения.
func NewAMQPBroker(url string) *AMQPBroker {
	return &AMQPBroker{url: url}
}

// Connect подключается к RabbitMQ и открывает канал.
func (b *AMQPBroker) Connect(_ context.Context) error {
	var err error
	b.conn, err = amqp.Dial(b.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	b.ch, err = b.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	log.Println("Connected to RabbitMQ")
	return nil
}

// DeclareQueue объявляет durable очередь.
func (b *AMQPBroker) DeclareQueue(_ context.Context, name string) error {
	if b.ch == nil {
		return fmt.Errorf("broker is not connected")
	}

	_, err := b.ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}

// Publish отправляет сообщение.
func (b *AMQPBroker) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	if b.ch == nil {
		return fmt.Errorf("broker is not connected")
	}

	return b.ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Consume возвращает канал сообщений для ручной обработки.
func (b *AMQPBroker) Consume(_ context.Context, queue string) (<-chan Delivery, error) {
	if b.ch == nil {
		return nil, fmt.Errorf("broker is not connected")
	}

	// autoAck=false: ручное подтверждение
	msgs, err := b.ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	out := make(chan Delivery)
	go func() {
		defer close(out)
		for d := range msgs {
			out <- Delivery{
				Body: d.Body,
				ACK:  func() error { return d.Ack(false) },
				NACK: func() error { return d.Nack(false, true) },
			}
		}
	}()

	return out, nil
}

// Close закрывает канал и соединение.
func (b *AMQPBroker) Close() error {
	var errs []error

	if b.ch != nil {
		if err := b.ch.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// DeclareExchange создаёт durable topic-exchange.
func (b *AMQPBroker) DeclareExchange(_ context.Context, name string) error {
	if b.ch == nil {
		return fmt.Errorf("broker is not connected")
	}
	return b.ch.ExchangeDeclare(name, "topic", true, false, false, false, nil)
}

// BindQueue привязывает очередь к exchange.
func (b *AMQPBroker) BindQueue(_ context.Context, queue, exchange, routingKey string) error {
	if b.ch == nil {
		return fmt.Errorf("broker is not connected")
	}
	return b.ch.QueueBind(queue, routingKey, exchange, false, nil)
}

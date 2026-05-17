package broker

import "context"

// Delivery представляет сообщение, полученное из очереди.
// ACK/NACK функции позволяют явно подтвердить или отклонить обработку.
type Delivery struct {
	Body []byte
	ACK  func() error
	NACK func() error
}

// Broker определяет контракт для взаимодействия с брокером сообщений.
type Broker interface {
	// Connect устанавливает соединение и открывает канал.
	Connect(ctx context.Context) error

	// DeclareQueue создаёт или проверяет существование очереди.
	DeclareQueue(ctx context.Context, name string) error

	// Publish отправляет сообщение в exchange с указанным routing key.
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error

	// Consume запускает чтение сообщений из очереди и возвращает канал Deliveries.
	Consume(ctx context.Context, queue string) (<-chan Delivery, error)

	// DeclareExchange создаёт exchange типа "topic".
	DeclareExchange(ctx context.Context, name string) error

	// BindQueue привязывает очередь к exchange с указанным routing key.
	BindQueue(ctx context.Context, queue, exchange, routingKey string) error

	// Close корректно закрывает канал и соединение.
	Close() error
}

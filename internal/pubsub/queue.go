package pubsub

import amqp "github.com/rabbitmq/amqp091-go"

const (
	Transient int = iota
	Durable
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	q, err := ch.QueueDeclare(queueName, simpleQueueType == Durable, simpleQueueType == Transient, simpleQueueType == Transient, false, nil)
	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	return ch, q, nil
}

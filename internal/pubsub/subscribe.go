package pubsub

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType string

const (
	Ack         AckType = "Ack"
	NackRequeue AckType = "NackRequeue"
	NackDiscard AckType = "NackDiscard"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	delivery, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for data := range delivery {
			var raw T
			json.Unmarshal(data.Body, &raw)
			switch handler(raw) {
			case Ack:
				log.Println("Ack")
				data.Ack(false)
			case NackRequeue:
				log.Println("Nack Requeue")
				data.Nack(false, true)
			case NackDiscard:
				log.Println("Nack Discard")
				data.Nack(false, false)
			}
		}
	}()

	return nil
}

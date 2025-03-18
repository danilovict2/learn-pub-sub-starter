package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"

	ampq "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *ampq.Channel, exchange, key string, val T) error {
	json, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, ampq.Publishing{
		ContentType: "application/json",
		Body:        json,
	})
}

func PublishGob[T any](ch *ampq.Channel, exchange, key string, val T) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(val); err != nil {
		return err
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, ampq.Publishing{
		ContentType: "application/gob",
		Body:        b.Bytes(),
	})
}

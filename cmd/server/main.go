package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	ampq "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"
	conn, err := ampq.Dial(rabbitConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Peril game server connected to RabbitMQ!")

	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, "game_logs.*", pubsub.Durable)
	if err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()
Loop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			log.Println("Pausing")
			publishPause(conn, true)
		case "resume":
			log.Println("Resuming")
			publishPause(conn, false)
		case "quit":
			log.Println("Quitting")
			break Loop
		default:
			log.Println("Don't understand the command:", words[0])
		
		}
	}
}

func publishPause(conn *ampq.Connection, isPaused bool) {
	ampqChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	if err := pubsub.PublishJSON(ampqChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: isPaused,
	}); err != nil {
		log.Fatal(err)
	}
}

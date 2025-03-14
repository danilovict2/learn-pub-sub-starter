package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Peril game client connected to RabbitMQ!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, "pause."+username, routing.PauseKey, pubsub.Transient)
	if err != nil {
		log.Fatal(err)
	}

	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(conn, string(routing.ExchangePerilDirect), "pause."+username, string(routing.PauseKey), pubsub.Transient, handlerPause(gameState))
	if err != nil {
		log.Fatal(err)
	}

Loop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			log.Println("Spawning Units")
			if err := gameState.CommandSpawn(words); err != nil {
				log.Fatal(err)
			}

		case "move":
			log.Println("Moving Units")
			if _, err := gameState.CommandMove(words); err != nil {
				log.Fatal(err)
			}
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet!")
		case "quit":
			log.Println("Quitting")
			break Loop
		default:
			log.Println("Don't understand the command:", words[0])

		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	defer fmt.Print("> ")
	return func(ps routing.PlayingState) {
		gs.HandlePause(ps)
	}
}

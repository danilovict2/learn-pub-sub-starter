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

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, "pause."+username, routing.PauseKey, pubsub.Transient, handlerPause(gameState))
	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "army_moves."+username, "army_moves.*", pubsub.Transient, handlerMove(gameState))
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
			move, err := gameState.CommandMove(words)
			if err != nil {
				log.Fatal(err)
			}

			publishMove(conn, move, username)
			
			log.Println("Move published successfully")
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
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(move)
	}
}

func publishMove(conn *amqp.Connection, move gamelogic.ArmyMove, username string) {
	ampqChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	if err := pubsub.PublishJSON(ampqChan, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, move); err != nil {
		log.Fatal(err)
	}
}
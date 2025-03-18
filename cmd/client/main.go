package main

import (
	"fmt"
	"log"
	"time"

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

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

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

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "army_moves."+username, "army_moves.*", pubsub.Transient, handlerMove(gameState, ch))
	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "war", "army_moves.*", pubsub.Durable, hadlerWar(gameState, ch))
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)

		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.Player.Username,
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				})
			if err != nil {
				fmt.Printf("could not publish war recognition: %v", err)
				return pubsub.NackRequeue
			}

			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func hadlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(row gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(row)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.Player.Username,
				Message:     "was not involved in the war",
			}
			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.Player.Username, log)

			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.Player.Username,
				Message:     "had no units to fight",
			}

			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.Player.Username, log)
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeYouWon:
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.Player.Username,
				Message:     fmt.Sprintf("%s won a war against %s", row.Attacker.Username, row.Defender.Username),
			}
			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.Player.Username, log)
			return pubsub.Ack
		case gamelogic.WarOutcomeOpponentWon:
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.Player.Username,
				Message:     fmt.Sprintf("%s won a war against %s", row.Defender.Username, row.Attacker.Username),
			}

			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.Player.Username, log)
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.Player.Username,
				Message:     fmt.Sprintf("A war between %s and %s resulted in a draw", row.Attacker.Username, row.Defender.Username),
			}

			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.Player.Username, log)
			return pubsub.Ack
		default:
			log.Println("Unrecognized outcome:", outcome)
			return pubsub.NackDiscard
		}
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

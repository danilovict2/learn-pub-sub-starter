package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

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
	
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)	
	<-signalChan
}

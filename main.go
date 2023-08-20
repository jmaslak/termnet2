package main

import (
	"fmt"
	"log"

	"github.com/jmaslak/termnet2/connector"
)

func main() {
	fmt.Println("Starting...")
	t := connector.NewTcp()

	_, notify, err := t.Listen(":2222")
	if err != nil {
		log.Fatal(err)
	}

	loop := connector.NewLoop()
	for {
		m := <-notify
		switch m.(type) {
		case connector.NewConnectionMessage:
			msg := m.(connector.NewConnectionMessage)
			fmt.Println("New connection!")
			loop.Dial(msg.Out, msg.In)
		default:
			log.Fatal("Unknown message type: " + m.Type())
		}
	}
}

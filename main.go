package main

import (
	"fmt"
	"log"

	"github.com/jmaslak/termnet2/connector"
)

func main() {
	nodeId := "0"
	fmt.Println("Starting...")
	listen, err := connector.NewTcpListen(nodeId, ":2222")
	if err != nil {
		log.Fatal(err)
	}

	for {
		m := <-listen.Notify()
		switch m.(type) {
		case connector.NewConnectionMessage:
			msg := m.(connector.NewConnectionMessage)
			fmt.Println("New connection!")
			telnetConn, err := connector.NewTelnetFilter(msg.Conn)
			if err != nil {
				log.Fatal(err)
			}
			filteredConn, err := connector.NewNewlineOutFilter(telnetConn)
			if err != nil {
				log.Fatal(err)
			}
			connector.StartLoopApp(filteredConn)
		default:
			log.Fatal("Unknown message type: " + m.TypeString())
		}
	}
}

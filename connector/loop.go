package connector

import (
	"log"
)

type loopApp struct {
	id   string
	conn Connection
}

func StartLoopApp(conn Connection) {
	loop := loopApp{}

	// Set up defaults
	loop.id = conn.Id() + "-LoopApp"
	loop.conn = conn

	go loop.repeat()
}

func (loop loopApp) Id() string { return loop.id }

func (loop loopApp) repeat() {
	in := loop.conn.FromConn()
	out := loop.conn.ToConn()
	defer close(out)

	for {
		m, ok := <-in
		if !ok {
			log.Print("Input channel closed.")
			return
		}
		switch m.(type) {
		case TextMessage:
			out <- m
		case DisconnectMessage:
			// We just disconnect
			log.Print("Disconnect received in loop connector")
			return
		default:
			log.Print("Unknown message type: " + m.Type())
		}
	}
}

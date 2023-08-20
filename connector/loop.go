package connector

import (
	"log"
)

type loopConn struct {
	Id string
}

func NewLoop() loopConn {
	loop := loopConn{}

	// Set up defaults
	loop.Id = "Loop"

	return loop
}

func (loop loopConn) Dial(in chan message, out chan message) error {
	go loop.repeat(in, out)
	return nil
}

func (loop loopConn) repeat(in chan message, out chan message) {
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
